"""Stasharr Tagger: tells Stasharr what is happening in video frames.

Stasharr sends frames from a scene and a list of descriptions (one per
marker label). The tagger scores how well each frame matches each
description with an OpenCLIP model and sends the scores back. Stasharr
turns runs of frames into markers.

Runs on the CPU, or on an NVIDIA card when one is passed to the container.
"""

import base64
import io
import os
import threading
import time
from typing import List, Optional

import torch
import open_clip
from fastapi import FastAPI, HTTPException
from PIL import Image
from pydantic import BaseModel

MODEL = os.environ.get("TAGGER_MODEL", "ViT-B-32")
PRETRAINED = os.environ.get("TAGGER_PRETRAINED", "laion2b_s34b_b79k")
THREADS = int(os.environ.get("TAGGER_THREADS", "2"))
WANT = os.environ.get("TAGGER_DEVICE", "auto").lower()
MAX_IMAGES = 64

# keep the rest of the server responsive
try:
    os.nice(int(os.environ.get("TAGGER_NICE", "10")))
except OSError:
    pass
torch.set_num_threads(max(1, THREADS))

if WANT == "cpu":
    DEVICE = "cpu"
elif WANT in ("cuda", "gpu"):
    DEVICE = "cuda"
else:
    DEVICE = "cuda" if torch.cuda.is_available() else "cpu"

app = FastAPI(title="Stasharr Tagger")

_lock = threading.Lock()
_state = {"model": None, "preprocess": None, "tokenizer": None, "error": None, "loading": False}
_text_cache: dict = {}


def _load():
    with _lock:
        if _state["model"] is not None or _state["error"] is not None:
            return
        _state["loading"] = True
        try:
            model, _, preprocess = open_clip.create_model_and_transforms(MODEL, pretrained=PRETRAINED, device=DEVICE)
            model.eval()
            if DEVICE == "cuda":
                model = model.half()
            _state["model"] = model
            _state["preprocess"] = preprocess
            _state["tokenizer"] = open_clip.get_tokenizer(MODEL)
        except Exception as e:  # noqa: BLE001 - reported on /health
            _state["error"] = str(e)
        finally:
            _state["loading"] = False


# load in the background so /health answers straight away
threading.Thread(target=_load, daemon=True).start()


def _ready():
    if _state["error"]:
        raise HTTPException(503, f"model failed to load: {_state['error']}")
    if _state["model"] is None:
        _load()
    if _state["model"] is None:
        raise HTTPException(503, "model is still loading")


def _text_features(labels: List[str]) -> torch.Tensor:
    """One vector per label. A label can hold several descriptions split by
    "|"; their vectors are averaged, which describes the label far better
    than any single sentence (prompt ensembling)."""
    key = (MODEL, PRETRAINED, tuple(labels))
    cached = _text_cache.get(key)
    if cached is not None:
        return cached
    parts, owner = [], []
    for i, label in enumerate(labels):
        pieces = [p.strip() for p in label.split("|") if p.strip()] or [label]
        parts.extend(pieces)
        owner.extend([i] * len(pieces))
    tokens = _state["tokenizer"](parts).to(DEVICE)
    with torch.no_grad():
        feats = _state["model"].encode_text(tokens)
        feats = feats / feats.norm(dim=-1, keepdim=True)
        index = torch.tensor(owner, device=feats.device)
        summed = torch.zeros(len(labels), feats.shape[1], device=feats.device, dtype=feats.dtype)
        summed.index_add_(0, index, feats)
        feats = summed / summed.norm(dim=-1, keepdim=True)
    if len(_text_cache) > 32:
        _text_cache.clear()
    _text_cache[key] = feats
    return feats


class ClassifyRequest(BaseModel):
    # descriptions to score, for example "a photo of two people kissing"
    labels: List[str]
    # JPEG or PNG frames, base64 encoded
    images: List[str]
    # also send back each frame's vector, so Stasharr can learn from the
    # markers an admin keeps or throws away
    vectors: bool = False


class ClassifyResponse(BaseModel):
    # one row per image, one score per label; each row adds up to 1
    scores: List[List[float]]
    seconds: float
    vectors: Optional[List[List[float]]] = None


@app.get("/health")
def health():
    return {
        "ok": _state["model"] is not None,
        "loading": _state["loading"] or (_state["model"] is None and _state["error"] is None),
        "error": _state["error"],
        "device": DEVICE,
        "gpu": torch.cuda.get_device_name(0) if DEVICE == "cuda" and torch.cuda.is_available() else None,
        "model": f"{MODEL} ({PRETRAINED})",
        "threads": THREADS,
        # what this version can do, so Stasharr knows what to send
        "features": ["ensemble", "vectors"],
    }


@app.post("/classify", response_model=ClassifyResponse)
def classify(req: ClassifyRequest):
    if not req.labels:
        raise HTTPException(400, "no labels")
    if not req.images:
        return ClassifyResponse(scores=[], seconds=0)
    if len(req.images) > MAX_IMAGES:
        raise HTTPException(400, f"send at most {MAX_IMAGES} images at a time")
    _ready()
    started = time.time()
    batch = []
    for b in req.images:
        try:
            img = Image.open(io.BytesIO(base64.b64decode(b))).convert("RGB")
        except Exception as e:  # noqa: BLE001
            raise HTTPException(400, f"unreadable image: {e}")
        batch.append(_state["preprocess"](img))
    pixels = torch.stack(batch).to(DEVICE)
    if DEVICE == "cuda":
        pixels = pixels.half()
    with _lock, torch.no_grad():
        text = _text_features(req.labels)
        img_feats = _state["model"].encode_image(pixels)
        img_feats = img_feats / img_feats.norm(dim=-1, keepdim=True)
        probs = (100.0 * img_feats @ text.T).float().softmax(dim=-1)
    vectors = None
    if req.vectors:
        vectors = [[round(x, 4) for x in r] for r in img_feats.float().cpu().tolist()]
    return ClassifyResponse(
        scores=[[round(float(x), 5) for x in row] for row in probs.cpu()],
        seconds=round(time.time() - started, 3),
        vectors=vectors,
    )


# ---------------------------------------------------------------- embeddings
# Stasharr keeps one vector per sampled frame so people can search scenes by
# describing them. Vectors are normalised, so a dot product is the match.


class EmbedImagesRequest(BaseModel):
    images: List[str]


class EmbedTextRequest(BaseModel):
    texts: List[str]


class EmbedResponse(BaseModel):
    vectors: List[List[float]]
    dim: int
    model: str


def _decode(images: List[str]):
    batch = []
    for b in images:
        try:
            img = Image.open(io.BytesIO(base64.b64decode(b))).convert("RGB")
        except Exception as e:  # noqa: BLE001
            raise HTTPException(400, f"unreadable image: {e}")
        batch.append(_state["preprocess"](img))
    return batch


@app.post("/embed_images", response_model=EmbedResponse)
def embed_images(req: EmbedImagesRequest):
    if len(req.images) > MAX_IMAGES:
        raise HTTPException(400, f"send at most {MAX_IMAGES} images at a time")
    _ready()
    if not req.images:
        return EmbedResponse(vectors=[], dim=0, model=f"{MODEL} ({PRETRAINED})")
    pixels = torch.stack(_decode(req.images)).to(DEVICE)
    if DEVICE == "cuda":
        pixels = pixels.half()
    with _lock, torch.no_grad():
        feats = _state["model"].encode_image(pixels)
        feats = feats / feats.norm(dim=-1, keepdim=True)
    rows = feats.float().cpu().tolist()
    return EmbedResponse(
        vectors=[[round(x, 5) for x in r] for r in rows], dim=len(rows[0]), model=f"{MODEL} ({PRETRAINED})"
    )


@app.post("/embed_text", response_model=EmbedResponse)
def embed_text(req: EmbedTextRequest):
    if not req.texts or len(req.texts) > 64:
        raise HTTPException(400, "send 1 to 64 texts")
    _ready()
    with _lock, torch.no_grad():
        tokens = _state["tokenizer"](req.texts).to(DEVICE)
        feats = _state["model"].encode_text(tokens)
        feats = feats / feats.norm(dim=-1, keepdim=True)
    rows = feats.float().cpu().tolist()
    return EmbedResponse(
        vectors=[[round(x, 5) for x in r] for r in rows], dim=len(rows[0]), model=f"{MODEL} ({PRETRAINED})"
    )
