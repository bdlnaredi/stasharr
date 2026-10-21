import React, { useEffect, useState } from "react";
import { Button, Form, Table } from "react-bootstrap";
import { faTrashAlt, faUndo } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { SettingRow, baseName, dirName, formatBytes, formatDate } from "./util";

export const TrashPanel: React.FC = () => {
  const Toast = useToast();
  const { data, loading, refetch } = GQL.useTrashBinQuery({
    fetchPolicy: "network-only",
  });
  const [configure] = GQL.useConfigureTrashMutation();
  const [restore] = GQL.useTrashRestoreMutation();
  const [destroy] = GQL.useTrashDeleteMutation();
  const [busy, setBusy] = useState(false);
  const [days, setDays] = useState<string>("");

  const cfg = data?.trashBin.config;
  useEffect(() => {
    if (cfg) setDays(String(cfg.retention_days));
  }, [cfg]);

  if (loading && !data) return <LoadingIndicator />;
  if (!data || !cfg) return null;

  const items = data.trashBin.items;

  async function run(fn: () => Promise<unknown>, success?: string) {
    setBusy(true);
    try {
      await fn();
      if (success) Toast.success(success);
      await refetch();
    } catch (e) {
      Toast.error(e);
    } finally {
      setBusy(false);
    }
  }

  function saveDays() {
    const n = Number.parseInt(days, 10);
    if (Number.isNaN(n) || n === cfg!.retention_days) {
      setDays(String(cfg!.retention_days));
      return;
    }
    run(() => configure({ variables: { input: { retention_days: n } } }));
  }

  function onRestore(item: GQL.TrashItemDataFragment) {
    run(
      () => restore({ variables: { ids: [item.id] } }),
      `Restored ${baseName(item.original_path)}. It is being scanned and identified again.`
    );
  }

  function onDelete(item: GQL.TrashItemDataFragment) {
    const name = baseName(item.original_path || item.trash_path);
    if (!window.confirm(`Delete ${name} for good? This cannot be undone.`)) {
      return;
    }
    run(() => destroy({ variables: { ids: [item.id] } }), `Deleted ${name}`);
  }

  function onEmpty() {
    if (
      !window.confirm(
        `Delete all ${items.length} item(s) in the trash bin for good? This cannot be undone.`
      )
    ) {
      return;
    }
    run(() => destroy({ variables: { ids: [] } }), "Trash bin emptied");
  }

  const location = cfg.trash_path
    ? `Files go to ${cfg.trash_path} (set under Settings → System).`
    : "Files go to a .stasharr-trash folder at the top of each library folder, so deleting is instant and needs no extra space.";

  return (
    <div className="stasharr-trash">
      <div className="setting-section">
        <h1>Trash bin</h1>
        <div className="sub-heading">
          Files you delete from the library are moved here first, so a mistake
          can be undone. Files in the trash bin are never scanned.
        </div>

        <SettingRow
          heading="Use the trash bin"
          sub={
            cfg.enabled
              ? location
              : "Off. Deleted files are removed straight away, unless a trash folder is set under Settings → System."
          }
        >
          <Form.Switch
            id="trash-enabled"
            disabled={busy}
            checked={cfg.enabled}
            onChange={() =>
              run(() =>
                configure({ variables: { input: { enabled: !cfg.enabled } } })
              )
            }
          />
        </SettingRow>

        <SettingRow
          heading="Keep files for (days)"
          sub="After this, files are deleted for good. 0 keeps them until you empty the bin."
        >
          <Form.Control
            className="text-input stasharr-number-input"
            type="number"
            min={0}
            max={3650}
            value={days}
            disabled={busy}
            onChange={(e) => setDays(e.currentTarget.value)}
            onBlur={() => saveDays()}
            onKeyDown={(e: React.KeyboardEvent) => {
              if (e.key === "Enter") saveDays();
            }}
          />
        </SettingRow>
      </div>

      <div className="setting-section">
        <div className="stasharr-section-head">
          <h1>
            In the trash bin{" "}
            <small className="text-muted">
              {items.length} item(s), {formatBytes(data.trashBin.total_size)}
            </small>
          </h1>
          <Button
            variant="danger"
            disabled={busy || items.length === 0}
            onClick={() => onEmpty()}
          >
            Empty trash bin
          </Button>
        </div>

        {items.length === 0 ? (
          <div className="sub-heading">The trash bin is empty.</div>
        ) : (
          <Table responsive size="sm" className="stasharr-table">
            <thead>
              <tr>
                <th>File</th>
                <th>Size</th>
                <th>Deleted</th>
                <th>Deleted for good</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id}>
                  <td className="stasharr-path-cell">
                    <div>
                      {baseName(item.original_path || item.trash_path)}
                      {item.is_dir && " (folder)"}
                    </div>
                    <small className="text-muted">
                      {item.original_path
                        ? `from ${dirName(item.original_path)}`
                        : "Original folder not known"}
                    </small>
                  </td>
                  <td>{formatBytes(item.size)}</td>
                  <td>{formatDate(item.deleted_at)}</td>
                  <td>{item.purge_at ? formatDate(item.purge_at) : "Never"}</td>
                  <td className="text-right text-nowrap">
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={busy || !item.original_path}
                      title="Put the file back and import it again"
                      onClick={() => onRestore(item)}
                    >
                      <Icon icon={faUndo} /> Restore
                    </Button>{" "}
                    <Button
                      variant="danger"
                      size="sm"
                      disabled={busy}
                      title="Delete for good"
                      onClick={() => onDelete(item)}
                    >
                      <Icon icon={faTrashAlt} />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
        <div className="sub-heading">
          Restored files are scanned and identified again, like a new
          download. Ratings, play history and your own edits of the deleted
          scene are not brought back.
        </div>
      </div>
    </div>
  );
};

export default TrashPanel;
