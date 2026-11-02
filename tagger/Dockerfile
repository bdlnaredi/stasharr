# Stasharr Tagger. Build with --build-arg VARIANT=cpu (default) or cuda.
FROM python:3.12-slim

ARG VARIANT=cpu
ENV PIP_NO_CACHE_DIR=1 \
    PYTHONUNBUFFERED=1 \
    HF_HOME=/models \
    TAGGER_THREADS=2

# PyTorch first: the CPU build is small, the CUDA build brings its own
# NVIDIA libraries (the host only needs the driver)
RUN if [ "$VARIANT" = "cuda" ]; then \
        pip install torch==2.6.0 torchvision==0.21.0 --index-url https://download.pytorch.org/whl/cu124; \
    else \
        pip install torch==2.6.0 torchvision==0.21.0 --index-url https://download.pytorch.org/whl/cpu; \
    fi

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY app.py .

# the model is downloaded here on first start; keep it in a volume
VOLUME /models
EXPOSE 2705
HEALTHCHECK --interval=60s --timeout=5s --start-period=600s CMD python -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:2705/health')"
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "2705"]
