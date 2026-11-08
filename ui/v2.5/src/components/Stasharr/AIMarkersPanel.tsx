import React, { useEffect, useMemo, useState } from "react";
import { Badge, Button, Form } from "react-bootstrap";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { SettingRow, baseName } from "./util";
import { TagClipsSection } from "./TagClipsSection";
import { SearchIndexSection } from "./Describe";

type Cfg = GQL.AiMarkerConfigDataFragment;
type Label = Cfg["labels"][number];
type Suggestion = GQL.AiMarkerSuggestionsQuery["aiMarkerSuggestions"]["suggestions"][number];

function clock(sec: number): string {
  const s = Math.max(0, Math.round(sec));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const r = s % 60;
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${String(r).padStart(2, "0")}` : `${m}:${String(r).padStart(2, "0")}`;
}

// ---------------------------------------------------------------- status

const TaggerStatus: React.FC<{ onRun: () => void; hasURL: boolean }> = ({ onRun, hasURL }) => {
  const { data, loading, refetch } = GQL.useAiTaggerStatusQuery({ fetchPolicy: "network-only" });
  const st = data?.aiTaggerStatus;

  useEffect(() => {
    // keep checking while the model loads
    if (!st?.loading) return;
    const timer = window.setTimeout(() => refetch(), 10000);
    return () => window.clearTimeout(timer);
  }, [st, refetch]);

  let badge = <Badge variant="secondary">Not set up</Badge>;
  if (st?.ok) badge = <Badge variant="success">Ready</Badge>;
  else if (st?.loading) badge = <Badge variant="warning">Loading the model…</Badge>;
  else if (st?.reachable) badge = <Badge variant="danger">Model failed</Badge>;
  else if (hasURL) badge = <Badge variant="danger">Not reachable</Badge>;

  return (
    <div className="setting-section">
      <h1>Stasharr Tagger</h1>
      <div className="sub-heading">
        A separate container that looks at frames from your scenes and recognises what is happening, then
        suggests markers. Run <code>ghcr.io/bdlnaredi/stasharr-tagger:latest</code> on the CPU, or{" "}
        <code>:cuda</code> with an NVIDIA card. The first start downloads the model, which takes a few minutes.
      </div>
      <SettingRow heading="Status">
        <div className="stasharr-ai-status">
          {loading && !st ? "Checking…" : badge}
          {st?.device && (
            <span>
              {st.device === "cuda" ? `NVIDIA: ${st.gpu ?? "graphics card"}` : "CPU"}
              {st.model ? ` · ${st.model}` : ""}
            </span>
          )}
          {st?.error && !st.ok && <span className="text-muted">{st.error}</span>}
          <Button size="sm" variant="secondary" onClick={() => refetch()}>
            Check again
          </Button>
        </div>
      </SettingRow>
      {st && (
        <SettingRow
          heading="Scenes looked at"
          sub="New scenes are looked at as they are imported when that is turned on below."
        >
          <div className="stasharr-ai-status">
            <span>
              {st.done.toLocaleString()} of {st.total.toLocaleString()}
            </span>
            <Button size="sm" variant="primary" disabled={!st.ok || st.done >= st.total} onClick={onRun}>
              Look at the rest
            </Button>
          </div>
        </SettingRow>
      )}
    </div>
  );
};

// ---------------------------------------------------------------- labels

const LabelsEditor: React.FC<{ cfg: Cfg; save: (input: GQL.AiMarkerConfigInput, msg?: string) => Promise<void> }> = ({
  cfg,
  save,
}) => {
  const strip = (ls: readonly Label[]) => ls.map((l) => ({ name: l.name, prompt: l.prompt, enabled: l.enabled, marker: l.marker }));
  const [labels, setLabels] = useState(strip(cfg.labels));
  useEffect(() => setLabels(strip(cfg.labels)), [cfg.labels]);
  const dirty = JSON.stringify(labels) !== JSON.stringify(strip(cfg.labels));

  function set(i: number, change: Partial<Label>) {
    setLabels(labels.map((l, j) => (j === i ? { ...l, ...change } : l)));
  }

  return (
    <div className="setting-section">
      <h1>What to look for</h1>
      <div className="sub-heading">
        Each label becomes a marker title and tag. The description is what the model compares frames with, so
        plain, visual wording works best. Labels that do not make markers help the model ignore talking, title
        cards and empty rooms.
      </div>
      <div className="stasharr-ai-labels">
        <div className="stasharr-ai-label head">
          <span>On</span>
          <span>Name</span>
          <span>Description for the model</span>
          <span>Marker</span>
          <span />
        </div>
        {labels.map((l, i) => (
          <div className={`stasharr-ai-label ${l.enabled ? "" : "off"}`} key={i}>
            <Form.Check checked={l.enabled} onChange={() => set(i, { enabled: !l.enabled })} />
            <Form.Control className="text-input" value={l.name} onChange={(e) => set(i, { name: e.currentTarget.value })} />
            <Form.Control className="text-input" value={l.prompt} onChange={(e) => set(i, { prompt: e.currentTarget.value })} />
            <Form.Check
              title="Make markers for this label"
              checked={l.marker}
              onChange={() => set(i, { marker: !l.marker })}
            />
            <Button size="sm" variant="danger" title="Remove" onClick={() => setLabels(labels.filter((_, j) => j !== i))}>
              ✕
            </Button>
          </div>
        ))}
      </div>
      <div className="stasharr-ai-actions">
        <Button
          size="sm"
          variant="secondary"
          onClick={() => setLabels([...labels, { name: "", prompt: "a photo of ", enabled: true, marker: true }])}
        >
          Add a label
        </Button>
        <Button
          size="sm"
          variant="secondary"
          onClick={() => {
            if (window.confirm("Go back to the original list of labels?")) save({ labels: [] }, "Labels reset");
          }}
        >
          Original labels
        </Button>
        {dirty && (
          <Button size="sm" variant="primary" onClick={() => save({ labels }, "Labels saved")}>
            Save labels
          </Button>
        )}
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- review

const MomentPreview: React.FC<{ s: Suggestion }> = ({ s }) => {
  const [play, setPlay] = useState(false);
  const src = s.scene.paths.stream ? `${s.scene.paths.stream}#t=${Math.floor(s.seconds)}` : undefined;
  return (
    <div className="stasharr-ai-moment">
      {play && src ? (
        <video src={src} autoPlay muted controls playsInline />
      ) : (
        <button type="button" onClick={() => setPlay(true)} title="Play this moment">
          {s.scene.paths.screenshot && <img src={s.scene.paths.screenshot} alt="" loading="lazy" />}
          <span>▶ {clock(s.seconds)}</span>
        </button>
      )}
    </div>
  );
};

const ReviewList: React.FC = () => {
  const Toast = useToast();
  const { data, loading, refetch } = GQL.useAiMarkerSuggestionsQuery({
    variables: { page: 1, per_page: 150 },
    fetchPolicy: "network-only",
  });
  const [approve] = GQL.useAiMarkerApproveMutation();
  const [reject] = GQL.useAiMarkerRejectMutation();
  const [busy, setBusy] = useState(false);

  const res = data?.aiMarkerSuggestions;
  const groups = useMemo(() => {
    const map = new Map<string, Suggestion[]>();
    for (const s of res?.suggestions ?? []) {
      const list = map.get(s.scene.id) ?? [];
      list.push(s);
      map.set(s.scene.id, list);
    }
    return [...map.values()];
  }, [res]);

  async function act(kind: "approve" | "reject", ids: string[]) {
    setBusy(true);
    try {
      if (kind === "approve") {
        const r = await approve({ variables: { ids } });
        Toast.success(`${r.data?.aiMarkerApprove ?? 0} marker(s) saved`);
      } else {
        await reject({ variables: { ids } });
      }
      await refetch();
    } catch (e) {
      Toast.error(e);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="setting-section">
      <h1>Waiting for review</h1>
      <div className="sub-heading">
        {res
          ? `${res.count.toLocaleString()} suggested marker(s) in ${res.scenes.toLocaleString()} scene(s). Keep the right ones; throw away the rest.`
          : "Suggested markers appear here."}
      </div>
      {loading && !res && <LoadingIndicator />}
      {groups.length === 0 && res && <p className="text-muted">Nothing waiting.</p>}
      {groups.map((list) => {
        const scene = list[0].scene;
        const ids = list.map((s) => s.id);
        return (
          <div className="stasharr-ai-scene" key={scene.id}>
            <div className="stasharr-ai-scene-head">
              <Link to={`/scenes/${scene.id}`}>{scene.title || baseName(scene.files[0]?.path ?? "") || `Scene ${scene.id}`}</Link>
              <div>
                <Button size="sm" variant="success" disabled={busy} onClick={() => act("approve", ids)}>
                  Keep all
                </Button>{" "}
                <Button size="sm" variant="secondary" disabled={busy} onClick={() => act("reject", ids)}>
                  Throw all away
                </Button>
              </div>
            </div>
            <div className="stasharr-ai-suggestions">
              {list.map((s) => (
                <div className="stasharr-ai-suggestion" key={s.id}>
                  <MomentPreview s={s} />
                  <div className="stasharr-ai-suggestion-text">
                    <b>{s.label}</b>
                    <span>
                      {clock(s.seconds)} – {clock(s.end_seconds)} · {Math.round(s.score * 100)}% sure
                    </span>
                  </div>
                  <div className="stasharr-ai-suggestion-actions">
                    <Button size="sm" variant="success" disabled={busy} onClick={() => act("approve", [s.id])} title="Keep">
                      ✓
                    </Button>
                    <Button size="sm" variant="secondary" disabled={busy} onClick={() => act("reject", [s.id])} title="Throw away">
                      ✕
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
};

// ---------------------------------------------------------------- panel

export const AIMarkersPanel: React.FC = () => {
  const Toast = useToast();
  const { data, loading, refetch } = GQL.useAiMarkerConfigQuery({ fetchPolicy: "network-only" });
  const [configure] = GQL.useConfigureAiMarkersMutation();
  const [run] = GQL.useRunAiMarkersMutation();
  const [url, setURL] = useState("");
  const cfg = data?.aiMarkerConfig;
  useEffect(() => setURL(cfg?.url ?? ""), [cfg?.url]);

  if (loading || !cfg) return <LoadingIndicator />;

  async function save(input: GQL.AiMarkerConfigInput, msg?: string) {
    try {
      await configure({ variables: { input } });
      await refetch();
      if (msg) Toast.success(msg);
    } catch (e) {
      Toast.error(e);
    }
  }

  async function runRest() {
    try {
      await run({ variables: {} });
      Toast.success("Looking at scenes. See Settings → Tasks for progress.");
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="stasharr-ai-panel">
      <div className="setting-section">
        <h1>AI markers</h1>
        <SettingRow
          heading="Tagger address"
          sub="Where Stasharr finds the tagger, for example http://stasharr-tagger:2705 or http://192.168.0.10:2705. Leave empty to turn AI markers off."
        >
          <div className="stasharr-ai-url">
            <Form.Control className="text-input" value={url} placeholder="http://stasharr-tagger:2705" onChange={(e) => setURL(e.currentTarget.value)} />
            <Button variant="primary" disabled={url === cfg.url} onClick={() => save({ url }, "Tagger address saved")}>
              Save
            </Button>
          </div>
        </SettingRow>
        <SettingRow heading="Look at new scenes" sub="When auto-import adds a scene, look at it after the reels are made.">
          <Form.Switch id="ai-auto" checked={cfg.auto} onChange={() => save({ auto: !cfg.auto })} />
        </SettingRow>
        <SettingRow
          heading="Review before saving"
          sub="On: suggestions wait below for you to keep or throw away. Off: they become markers straight away."
        >
          <Form.Switch id="ai-review" checked={cfg.review} onChange={() => save({ review: !cfg.review })} />
        </SettingRow>
        <SettingRow heading="Look every" sub="Seconds between frames. Fewer frames is faster; more is more exact.">
          <Form.Control
            as="select"
            className="input-control"
            value={String(cfg.interval)}
            onChange={(e) => save({ interval: Number(e.currentTarget.value) })}
          >
            {[2, 3, 4, 5, 8, 10].map((n) => (
              <option key={n} value={n}>
                {n} seconds
              </option>
            ))}
          </Form.Control>
        </SettingRow>
        <SettingRow heading="How sure" sub="Higher gives fewer, more reliable markers.">
          <Form.Control
            as="select"
            className="input-control"
            value={String(cfg.min_confidence)}
            onChange={(e) => save({ min_confidence: Number(e.currentTarget.value) })}
          >
            {[0.25, 0.3, 0.35, 0.45, 0.55, 0.7].map((n) => (
              <option key={n} value={n}>
                {Math.round(n * 100)}%
              </option>
            ))}
          </Form.Control>
        </SettingRow>
        <SettingRow heading="Shortest marker" sub="Something must go on this long to get a marker.">
          <Form.Control
            as="select"
            className="input-control"
            value={String(cfg.min_seconds)}
            onChange={(e) => save({ min_seconds: Number(e.currentTarget.value) })}
          >
            {[8, 12, 16, 24, 30, 45, 60].map((n) => (
              <option key={n} value={n}>
                {n} seconds
              </option>
            ))}
          </Form.Control>
        </SettingRow>
      </div>
      <TaggerStatus key={cfg.url} hasURL={!!cfg.url} onRun={runRest} />
      <ReviewList />
      <LabelsEditor cfg={cfg} save={save} />
      <TagClipsSection />
      <SearchIndexSection onToast={(m) => Toast.success(m)} onError={(e) => Toast.error(e)} />
    </div>
  );
};
