import React, { useEffect, useState } from "react";
import { Badge, Button, Form, Table } from "react-bootstrap";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { useMonitorJob } from "src/utils/job";
import { SettingRow } from "./util";

function statusBadge(status: string) {
  switch (status) {
    case "MOVE":
      return <Badge variant="primary">Rename</Badge>;
    case "SAME":
      return <Badge variant="secondary">Already right</Badge>;
    default:
      return <Badge variant="warning">Skip</Badge>;
  }
}

export const RenamerPanel: React.FC = () => {
  const Toast = useToast();
  const { data, loading, refetch } = GQL.useRenamerConfigQuery({
    fetchPolicy: "network-only",
  });
  const [configure] = GQL.useConfigureRenamerMutation();
  const [renameScenes] = GQL.useRenameScenesMutation();
  const [preview, previewResult] = GQL.useRenamePreviewLazyQuery({
    fetchPolicy: "network-only",
  });
  const [template, setTemplate] = useState("");
  const [saving, setSaving] = useState(false);
  const [jobID, setJobID] = useState<string>();

  const cfg = data?.renamerConfig;
  useEffect(() => {
    if (cfg) setTemplate(cfg.template);
  }, [cfg]);

  useMonitorJob(jobID, () => {
    setJobID(undefined);
    Toast.success("Renaming finished");
    if (previewResult.called) {
      preview({ variables: { input: { template } } });
    }
  });

  if (loading && !data) return <LoadingIndicator />;
  if (!cfg) return null;

  const dirty = template !== cfg.template;

  async function save(input: GQL.RenamerConfigInput) {
    setSaving(true);
    try {
      await configure({ variables: { input } });
      await refetch();
      Toast.success("Saved");
    } catch (e) {
      Toast.error(e);
    } finally {
      setSaving(false);
    }
  }

  function insertToken(token: string) {
    setTemplate((t) => t + token);
  }

  function onPreview() {
    preview({ variables: { input: { template } } });
  }

  async function onRenameAll() {
    const n = previewResult.data?.renamePreview.to_rename;
    const msg =
      n !== undefined
        ? `Rename and move ${n} file(s) using this template?`
        : "Rename and move every file in the library that has a title, using this template?";
    if (!window.confirm(msg)) return;
    try {
      const res = await renameScenes({ variables: { input: { template } } });
      setJobID(res.data?.renameScenes);
      Toast.success("Renaming files. See Settings → Tasks for progress.");
    } catch (e) {
      Toast.error(e);
    }
  }

  const result = previewResult.data?.renamePreview;

  return (
    <div className="stasharr-renamer">
      <div className="setting-section">
        <h1>File renamer</h1>
        <div className="sub-heading">
          Moves and renames video files using their metadata, inside the
          library folder they are already in. Subtitles and funscripts with the
          same name are moved too. Scenes without a title are left alone.
        </div>

        <SettingRow
          heading="Rename new scenes automatically"
          sub="After auto-import has identified a new scene, its file is renamed with the template below."
        >
          <Form.Switch
            id="renamer-enabled"
            disabled={saving}
            checked={cfg.enabled}
            onChange={() => save({ enabled: !cfg.enabled })}
          />
        </SettingRow>
      </div>

      <div className="setting-section">
        <h1>Template</h1>
        <div className="sub-heading">
          Use / to make folders. The file extension is added for you. Parts
          that are empty for a scene, such as a missing date, are left out
          along with their separators.
        </div>
        <Form.Group>
          <Form.Control
            className="text-input"
            value={template}
            spellCheck={false}
            onChange={(e) => setTemplate(e.currentTarget.value)}
          />
        </Form.Group>
        <div className="stasharr-token-list">
          {cfg.tokens.map((t) => (
            <Button
              key={t.token}
              size="sm"
              variant="secondary"
              title={t.description}
              onClick={() => insertToken(t.token)}
            >
              {t.token}
            </Button>
          ))}
        </div>
        <div className="stasharr-section-head">
          <div>
            <Button
              variant="primary"
              disabled={saving || !dirty}
              onClick={() => save({ template })}
            >
              Save template
            </Button>{" "}
            <Button
              variant="secondary"
              disabled={saving || template === cfg.default_template}
              onClick={() => setTemplate(cfg.default_template)}
            >
              Use default
            </Button>
          </div>
          <div>
            <Button
              variant="secondary"
              disabled={previewResult.loading}
              onClick={() => onPreview()}
            >
              Preview
            </Button>{" "}
            <Button
              variant="danger"
              disabled={!!jobID || (result !== undefined && result.to_rename === 0)}
              onClick={() => onRenameAll()}
            >
              Rename library
            </Button>
          </div>
        </div>
      </div>

      {previewResult.loading && <LoadingIndicator message="Working out new names…" />}
      {previewResult.error && (
        <div className="setting-section text-danger">
          {previewResult.error.message}
        </div>
      )}

      {result && !previewResult.loading && (
        <div className="setting-section">
          <h1>
            Preview{" "}
            <small className="text-muted">
              {result.to_rename} of {result.checked} file(s) would be renamed
              {result.items.length < result.checked &&
                ` (showing ${result.items.length})`}
            </small>
          </h1>
          <Table responsive size="sm" className="stasharr-table">
            <thead>
              <tr>
                <th />
                <th>Scene</th>
                <th>Now</th>
                <th>New name</th>
              </tr>
            </thead>
            <tbody>
              {result.items.map((it) => (
                <tr key={it.scene_id}>
                  <td>{statusBadge(it.status)}</td>
                  <td>
                    <Link to={`/scenes/${it.scene_id}`}>
                      {it.title || `Scene ${it.scene_id}`}
                    </Link>
                    {it.reason && (
                      <div>
                        <small className="text-muted">{it.reason}</small>
                      </div>
                    )}
                  </td>
                  <td className="stasharr-path">{it.old_path}</td>
                  <td className="stasharr-path">
                    {it.status === "MOVE" ? it.new_path : ""}
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </div>
      )}
    </div>
  );
};

export default RenamerPanel;
