import React, { useState } from "react";
import { Button, Form, Table } from "react-bootstrap";
import { faTrashAlt, faUndo } from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { SettingRow, formatBytes, formatDate } from "./util";

function hourLabel(h: number) {
  const suffix = h < 12 ? "am" : "pm";
  const n = h % 12 === 0 ? 12 : h % 12;
  return `${n}:00 ${suffix}`;
}

const NumberSetting: React.FC<{
  value: number;
  min: number;
  max: number;
  disabled: boolean;
  onSave: (n: number) => void;
}> = ({ value, min, max, disabled, onSave }) => {
  const [text, setText] = useState(String(value));
  function commit() {
    const n = Number.parseInt(text, 10);
    if (Number.isNaN(n) || n === value) {
      setText(String(value));
      return;
    }
    onSave(Math.max(min, Math.min(max, n)));
  }
  return (
    <Form.Control
      className="text-input stasharr-number-input"
      type="number"
      min={min}
      max={max}
      value={text}
      disabled={disabled}
      onChange={(e) => setText(e.currentTarget.value)}
      onBlur={() => commit()}
      onKeyDown={(e: React.KeyboardEvent) => {
        if (e.key === "Enter") commit();
      }}
    />
  );
};

export const MaintenancePanel: React.FC = () => {
  const Toast = useToast();
  const { data, loading, refetch } = GQL.useMaintenanceSettingsQuery({
    fetchPolicy: "network-only",
  });
  const [configure] = GQL.useConfigureMaintenanceMutation();
  const [backupNow] = GQL.useRunScheduledBackupMutation();
  const [restore] = GQL.useRestoreDatabaseBackupMutation();
  const [destroy] = GQL.useDeleteDatabaseBackupMutation();
  const [resyncNow] = GQL.useRunMetadataResyncMutation();
  const [busy, setBusy] = useState(false);

  if (loading && !data) return <LoadingIndicator />;
  if (!data) return null;
  const cfg = data.maintenanceConfig;
  const backups = data.databaseBackups;

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

  function set(input: GQL.MaintenanceConfigInput) {
    run(() => configure({ variables: { input } }));
  }

  function onRestore(b: (typeof backups)[number]) {
    if (
      !window.confirm(
        `Restore the backup from ${formatDate(b.created_at)}?\n\nEverything changed since then is lost. ` +
          "Running tasks are stopped, and the current database is backed up first so this can be undone."
      )
    ) {
      return;
    }
    run(
      () => restore({ variables: { id: b.id } }),
      "Backup restored. Reload the page to see the restored library."
    );
  }

  function onDelete(b: (typeof backups)[number]) {
    if (!window.confirm(`Delete the backup ${b.name}?`)) return;
    run(() => destroy({ variables: { id: b.id } }), "Backup deleted");
  }

  return (
    <div className="stasharr-maintenance">
      <div className="setting-section">
        <h1>Schedule</h1>
        <SettingRow
          heading="Run at"
          sub="Backups and metadata refreshes run at this time of day (server time), when the server is usually idle."
        >
          <Form.Control
            as="select"
            className="input-control"
            disabled={busy}
            value={cfg.hour}
            onChange={(e) => set({ hour: Number(e.currentTarget.value) })}
          >
            {Array.from({ length: 24 }, (_, h) => (
              <option key={h} value={h}>
                {hourLabel(h)}
              </option>
            ))}
          </Form.Control>
        </SettingRow>
      </div>

      <div className="setting-section">
        <div className="stasharr-section-head">
          <h1>Backups</h1>
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() =>
              run(() => backupNow(), "Backing up. See Settings → Tasks for progress.")
            }
          >
            Back up now
          </Button>
        </div>
        <SettingRow
          heading="Back up every day"
          sub={
            cfg.backup_last_run
              ? `Last backup started ${formatDate(cfg.backup_last_run)}. Saved in ${cfg.backup_dir}/stasharr-auto.`
              : `Saved in ${cfg.backup_dir}/stasharr-auto. Change the folder under Settings → System → Backup directory.`
          }
        >
          <Form.Switch
            id="backup-enabled"
            disabled={busy}
            checked={cfg.backup_enabled}
            onChange={() => set({ backup_enabled: !cfg.backup_enabled })}
          />
        </SettingRow>
        <SettingRow
          heading="Backups to keep"
          sub="Older scheduled backups are deleted. Backups you make from Settings → Tasks are never deleted."
        >
          <NumberSetting
            value={cfg.backup_keep}
            min={1}
            max={365}
            disabled={busy}
            onSave={(n) => set({ backup_keep: n })}
          />
        </SettingRow>
        <SettingRow
          heading="Include images"
          sub="Also back up covers and performer images stored on disk. Makes a much larger zip file."
        >
          <Form.Switch
            id="backup-blobs"
            disabled={busy}
            checked={cfg.backup_include_blobs}
            onChange={() => set({ backup_include_blobs: !cfg.backup_include_blobs })}
          />
        </SettingRow>

        {backups.length === 0 ? (
          <div className="sub-heading mt-3">No backups yet.</div>
        ) : (
          <Table responsive size="sm" className="stasharr-table mt-3">
            <thead>
              <tr>
                <th>Made</th>
                <th>Type</th>
                <th>Size</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {backups.map((b) => (
                <tr key={b.id}>
                  <td>
                    <div>{formatDate(b.created_at)}</div>
                    <small className="text-muted">{b.name}</small>
                  </td>
                  <td>
                    {b.scheduled ? "Scheduled" : "Manual"}
                    {b.includes_blobs && ", with images"}
                  </td>
                  <td>{formatBytes(b.size)}</td>
                  <td className="text-right text-nowrap">
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={busy}
                      onClick={() => onRestore(b)}
                    >
                      <Icon icon={faUndo} /> Restore
                    </Button>{" "}
                    <Button
                      variant="danger"
                      size="sm"
                      disabled={busy}
                      title="Delete backup"
                      onClick={() => onDelete(b)}
                    >
                      <Icon icon={faTrashAlt} />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
      </div>

      <div className="setting-section">
        <div className="stasharr-section-head">
          <h1>Metadata refresh</h1>
          <Button
            variant="secondary"
            disabled={busy || cfg.stash_boxes.length === 0}
            onClick={() =>
              run(() => resyncNow(), "Refreshing. See Settings → Tasks for progress.")
            }
          >
            Refresh now
          </Button>
        </div>
        <div className="sub-heading">
          {cfg.stash_boxes.length === 0
            ? "Add StashDB or ThePornDB under Settings → Metadata Providers first."
            : `Checks ${cfg.stash_boxes.join(", ")} again for scenes, performers and studios that are already matched, and adds new details, images, performers and tags. Scene titles and descriptions you have filled in are kept; performer and studio details are updated to match.`}
        </div>
        <SettingRow
          heading="Refresh automatically"
          sub={
            cfg.resync_last_run
              ? `Last refresh started ${formatDate(cfg.resync_last_run)}.`
              : "Never run yet."
          }
        >
          <Form.Switch
            id="resync-enabled"
            disabled={busy}
            checked={cfg.resync_enabled}
            onChange={() => set({ resync_enabled: !cfg.resync_enabled })}
          />
        </SettingRow>
        <SettingRow heading="Every (days)">
          <NumberSetting
            value={cfg.resync_interval_days}
            min={1}
            max={365}
            disabled={busy}
            onSave={(n) => set({ resync_interval_days: n })}
          />
        </SettingRow>
        <SettingRow heading="Refresh scenes">
          <Form.Switch
            id="resync-scenes"
            disabled={busy}
            checked={cfg.resync_scenes}
            onChange={() => set({ resync_scenes: !cfg.resync_scenes })}
          />
        </SettingRow>
        <SettingRow heading="Refresh performers" sub="Updated photos, aliases and details">
          <Form.Switch
            id="resync-performers"
            disabled={busy}
            checked={cfg.resync_performers}
            onChange={() => set({ resync_performers: !cfg.resync_performers })}
          />
        </SettingRow>
        <SettingRow heading="Refresh studios" sub="Updated logos and parent studios">
          <Form.Switch
            id="resync-studios"
            disabled={busy}
            checked={cfg.resync_studios}
            onChange={() => set({ resync_studios: !cfg.resync_studios })}
          />
        </SettingRow>
      </div>
    </div>
  );
};
