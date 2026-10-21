import React, { useState } from "react";
import { Badge, Button, Form } from "react-bootstrap";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { useMonitorJob } from "src/utils/job";
import {
  SettingRow,
  formatBitrate,
  formatBytes,
  formatDuration,
  resolutionLabel,
  sceneName,
} from "./util";

type DupeScene = GQL.DuplicateCleanupSceneDataFragment;

const accuracyOptions = [
  { value: 0, label: "Exact" },
  { value: 4, label: "High" },
  { value: 8, label: "Medium" },
  { value: 10, label: "Low" },
];

const durationOptions = [
  { value: 1, label: "Within 1 second" },
  { value: 5, label: "Within 5 seconds" },
  { value: 10, label: "Within 10 seconds" },
  { value: -1, label: "Any length" },
];

const DupeRow: React.FC<{ scene: DupeScene; keep: boolean }> = ({
  scene,
  keep,
}) => {
  const f = scene.files[0];
  return (
    <div className={`stasharr-dupe-scene ${keep ? "keep" : "remove"}`}>
      {scene.paths.screenshot && (
        <img src={scene.paths.screenshot} alt="" loading="lazy" />
      )}
      <div className="stasharr-dupe-info">
        <div>
          <Badge variant={keep ? "success" : "danger"}>
            {keep ? "Keep" : "Trash"}
          </Badge>{" "}
          <Link to={`/scenes/${scene.id}`}>{sceneName(scene)}</Link>
          {scene.studio && (
            <span className="text-muted"> · {scene.studio.name}</span>
          )}
        </div>
        {f && (
          <div className="stasharr-dupe-meta">
            {[
              resolutionLabel(f.width, f.height),
              formatBitrate(f.bit_rate),
              formatBytes(f.size),
              formatDuration(f.duration),
              f.video_codec.toUpperCase(),
              scene.organized ? "Organized" : "",
            ]
              .filter(Boolean)
              .join(" · ")}
          </div>
        )}
        {f && <small className="text-muted stasharr-path">{f.path}</small>}
      </div>
    </div>
  );
};

export const DuplicateCleanupPanel: React.FC = () => {
  const Toast = useToast();
  const [distance, setDistance] = useState(0);
  const [durationDiff, setDurationDiff] = useState(1);
  const [skipped, setSkipped] = useState<Set<string>>(new Set());
  const [jobID, setJobID] = useState<string>();

  const [find, { data, loading, refetch, called }] =
    GQL.useDuplicateCleanupPreviewLazyQuery({ fetchPolicy: "network-only" });
  const [cleanup] = GQL.useDuplicateCleanupMutation();

  useMonitorJob(jobID, () => {
    setJobID(undefined);
    Toast.success("Duplicate cleanup finished");
    refetch?.();
  });

  const input = { distance, duration_diff: durationDiff };
  const groups = data?.duplicateCleanupPreview ?? [];
  const selected = groups.filter((g) => !skipped.has(g.keep.id));
  const removeCount = selected.reduce((n, g) => n + g.remove.length, 0);
  const freed = selected.reduce(
    (n, g) => n + g.remove.reduce((m, s) => m + (s.files[0]?.size ?? 0), 0),
    0
  );

  function onFind() {
    setSkipped(new Set());
    find({ variables: { input } });
  }

  function toggle(id: string) {
    const next = new Set(skipped);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSkipped(next);
  }

  async function onCleanup() {
    if (
      !window.confirm(
        `Move ${removeCount} duplicate scene(s) to the trash bin? Their tags, performers, play history and markers are copied to the copy that is kept.`
      )
    ) {
      return;
    }
    try {
      const res = await cleanup({
        variables: {
          input: { ...input, keep_scene_ids: selected.map((g) => g.keep.id) },
        },
      });
      setJobID(res.data?.duplicateCleanup);
      Toast.success("Cleaning up duplicates. See Settings → Tasks for progress.");
    } catch (e) {
      Toast.error(e);
    }
  }

  return (
    <div className="stasharr-dupes">
      <div className="setting-section">
        <h1>Duplicate cleanup</h1>
        <div className="sub-heading">
          Finds copies of the same video and keeps the best one: highest
          resolution first, then highest bitrate, then largest file. Tags,
          performers, play history, ratings and markers of the other copies are
          copied to the one that is kept, and the other files go to the trash
          bin.
        </div>

        <SettingRow heading="How alike" sub="Exact is safest. Lower accuracy finds re-encodes and different cuts, but check the list before cleaning up.">
          <Form.Control
            as="select"
            className="input-control"
            value={distance}
            onChange={(e) => setDistance(Number(e.currentTarget.value))}
          >
            {accuracyOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Form.Control>
        </SettingRow>

        <SettingRow heading="Length" sub="How close in length duplicates must be.">
          <Form.Control
            as="select"
            className="input-control"
            value={durationDiff}
            onChange={(e) => setDurationDiff(Number(e.currentTarget.value))}
          >
            {durationOptions.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </Form.Control>
        </SettingRow>

        <div className="stasharr-section-head">
          <Button variant="secondary" disabled={loading} onClick={() => onFind()}>
            Find duplicates
          </Button>
          {groups.length > 0 && (
            <Button
              variant="danger"
              disabled={!!jobID || removeCount === 0}
              onClick={() => onCleanup()}
            >
              Clean up {selected.length} group(s), free {formatBytes(freed)}
            </Button>
          )}
        </div>
      </div>

      {loading && <LoadingIndicator message="Looking for duplicates…" />}

      {!loading && called && groups.length === 0 && (
        <div className="setting-section">
          <div className="sub-heading">No duplicates found.</div>
        </div>
      )}

      {!loading &&
        groups.map((g) => (
          <div className="setting-section stasharr-dupe-group" key={g.keep.id}>
            <Form.Check
              id={`dupe-group-${g.keep.id}`}
              label="Clean up this group"
              checked={!skipped.has(g.keep.id)}
              onChange={() => toggle(g.keep.id)}
            />
            <DupeRow scene={g.keep} keep />
            {g.remove.map((s) => (
              <DupeRow key={s.id} scene={s} keep={false} />
            ))}
          </div>
        ))}
    </div>
  );
};

export default DuplicateCleanupPanel;
