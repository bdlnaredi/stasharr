import React, { useState } from "react";
import { Badge, Button, Form, ProgressBar, Table } from "react-bootstrap";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { useToast } from "src/hooks/Toast";
import { SettingRow, formatBytes, formatDate } from "./util";

// Stasharr: Settings → Health. Disk space, the database and backups, the
// tagger, auto-import and failed tasks in one place.

const Check: React.FC<{ ok: boolean; warn?: boolean; label: string; detail?: React.ReactNode }> = ({
  ok,
  warn,
  label,
  detail,
}) => (
  <div className="stasharr-health-check">
    <span className={`stasharr-health-dot ${ok ? "ok" : warn ? "warn" : "bad"}`} />
    <div>
      <div>{label}</div>
      {detail && <div className="sub-heading">{detail}</div>}
    </div>
  </div>
);

export const HealthPanel: React.FC = () => {
  const Toast = useToast();
  const { data, refetch, loading } = GQL.useStasharrHealthQuery({ fetchPolicy: "network-only" });
  const [configure] = GQL.useConfigureHealthMutation();
  const [minFree, setMinFree] = useState<string>("");
  const h = data?.stasharrHealth;
  if (!h) return loading ? <div className="p-3">Checking…</div> : null;

  return (
    <div className="stasharr-health">
      <div className="setting-section">
        <div className="stasharr-section-head">
          <h1>Health</h1>
          <Button variant="secondary" onClick={() => refetch()}>
            Check again
          </Button>
        </div>
        {h.problems.length === 0 ? (
          <div className="stasharr-health-allgood">Everything looks fine.</div>
        ) : (
          <ul className="stasharr-health-problems">
            {h.problems.map((p, i) => (
              <li key={i} className={p.level}>
                {p.text}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="setting-section">
        <h1>Disk space</h1>
        {h.disks.map((d, i) => {
          const used = d.total > 0 ? ((d.total - d.free) / d.total) * 100 : 0;
          return (
            <div key={i} className="stasharr-health-disk">
              <div className="d-flex justify-content-between">
                <strong>{d.labels.join(", ")}</strong>
                <span className={d.low ? "text-danger" : ""}>
                  {formatBytes(d.free)} free of {formatBytes(d.total)}
                </span>
              </div>
              <ProgressBar now={used} variant={d.low ? "danger" : used > 85 ? "warning" : "info"} />
              <div className="sub-heading text-truncate">{d.paths.join(" · ")}</div>
            </div>
          );
        })}
        <SettingRow
          heading="Warn when free space is below"
          sub="GB, or under 5% of the disk. You get a notification (Settings → Notifications → Low disk space) at most every 12 hours."
        >
          <div className="d-flex">
            <Form.Control
              className="text-input mr-2"
              type="number"
              min={1}
              style={{ width: 100 }}
              value={minFree || String(h.min_free_gb)}
              onChange={(e) => setMinFree(e.currentTarget.value)}
            />
            <Button
              disabled={!minFree || Number(minFree) === h.min_free_gb}
              onClick={async () => {
                try {
                  await configure({ variables: { min_free_gb: Number(minFree) } });
                  setMinFree("");
                  await refetch();
                } catch (e) {
                  Toast.error(e);
                }
              }}
            >
              Save
            </Button>
          </div>
        </SettingRow>
      </div>

      <div className="setting-section">
        <h1>Services</h1>
        <Check
          ok={h.backups_enabled && !!h.last_backup}
          warn={!h.backups_enabled}
          label="Backups"
          detail={
            <>
              {h.backups_enabled ? "Daily backups on" : "Daily backups off"} · last backup {formatDate(h.last_backup)} ·
              database {formatBytes(h.database_size)}
              {h.drive_connected && (h.drive_last_error ? " · Google Drive upload failing" : " · copied to Google Drive")}
            </>
          }
        />
        <Check
          ok={!h.watcher_enabled || h.watcher_running}
          warn={!h.watcher_enabled}
          label="Auto-import"
          detail={
            h.watcher_enabled
              ? `${h.watcher_running ? "Watching for new files" : "Not running"} · last check ${formatDate(h.watcher_last_check)}`
              : "Off"
          }
        />
        <Check
          ok={!h.tagger_set || h.tagger_ok}
          warn={!h.tagger_set}
          label="Stasharr Tagger"
          detail={h.tagger_set ? h.tagger_detail ?? "" : "Not set up (Settings → AI markers)"}
        />
        <Check
          ok={h.notify_targets > 0}
          warn
          label="Notifications"
          detail={h.notify_targets > 0 ? `${h.notify_targets} place(s) get notifications` : "None set up"}
        />
        <div className="sub-heading mt-2">
          {h.scenes.toLocaleString()} scenes · Stasharr {h.version}
        </div>
      </div>

      <div className="setting-section">
        <h1>
          Failed tasks {h.failed_tasks.length > 0 && <Badge variant="danger">{h.failed_tasks.length}</Badge>}
        </h1>
        {h.failed_tasks.length === 0 ? (
          <div className="sub-heading">No task failed since the server started.</div>
        ) : (
          <Table className="stasharr-table" responsive size="sm">
            <tbody>
              {h.failed_tasks.map((f, i) => (
                <tr key={i}>
                  <td className="text-nowrap">{formatDate(f.at)}</td>
                  <td>
                    <div>{f.description}</div>
                    <small className="text-muted">{f.error}</small>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
        <div className="sub-heading">
          More detail is under <Link to="/settings?tab=logs">Logs</Link>.
        </div>
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- cleanup

const kinds: { kind: GQL.CleanupKind; label: string; sub: string }[] = [
  { kind: GQL.CleanupKind.Unwatched, label: "Not watched lately", sub: "Nobody played them in the chosen months, and they were added before that." },
  { kind: GQL.CleanupKind.Never, label: "Never watched", sub: "Never played by anyone, added over a month ago." },
  { kind: GQL.CleanupKind.LowRated, label: "Low rated", sub: "Two stars or less, from you or on average from everyone." },
];

export const CleanupPanel: React.FC = () => {
  const Toast = useToast();
  const [kind, setKind] = useState(GQL.CleanupKind.Unwatched);
  const [months, setMonths] = useState(6);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const trash = GQL.useTrashBinQuery({ fetchPolicy: "cache-first" });
  const { data, loading, refetch } = GQL.useCleanupSuggestionsQuery({
    variables: { kind, months, limit: 200 },
    fetchPolicy: "network-only",
  });
  const [destroy] = GQL.useScenesDestroyMutation();
  const res = data?.cleanupSuggestions;
  const items = res?.items ?? [];
  const trashOn = !!trash.data?.trashBin.config.enabled;
  const selectedSize = items.filter((i) => selected.has(i.scene.id)).reduce((s, i) => s + i.size, 0);

  function toggle(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
  }

  async function remove() {
    const ids = [...selected];
    const where = trashOn ? "moved to the trash bin" : "deleted for good";
    if (!window.confirm(`${ids.length} scene(s), ${formatBytes(selectedSize)}, will be ${where}. Go ahead?`)) return;
    try {
      await destroy({ variables: { ids, delete_file: true, delete_generated: true } });
      setSelected(new Set());
      await refetch();
      Toast.success(trashOn ? "Moved to the trash bin" : "Deleted");
    } catch (e) {
      Toast.error(e);
    }
  }

  const k = kinds.find((x) => x.kind === kind)!;
  return (
    <div className="setting-section stasharr-cleanup">
      <h1>Cleanup suggestions</h1>
      <div className="sub-heading">
        Scenes that take space without being watched or liked. Nothing is removed until you choose scenes and
        press Remove.{" "}
        {trashOn
          ? "Removed scenes go to the trash bin first, so they can be restored."
          : "The trash bin is off, so removed files are deleted for good (Settings → Trash bin)."}{" "}
        The same scene more than once is under <Link to="/settings?tab=duplicates">Duplicate cleanup</Link>.
      </div>
      <div className="stasharr-cleanup-tabs">
        {kinds.map((x) => (
          <Button
            key={x.kind}
            size="sm"
            variant={x.kind === kind ? "primary" : "secondary"}
            onClick={() => {
              setKind(x.kind);
              setSelected(new Set());
            }}
          >
            {x.label}
          </Button>
        ))}
        {kind === GQL.CleanupKind.Unwatched && (
          <Form.Control
            as="select"
            size="sm"
            className="input-control"
            style={{ width: "auto" }}
            value={months}
            onChange={(e) => setMonths(Number(e.currentTarget.value))}
          >
            {[3, 6, 12, 24].map((m) => (
              <option key={m} value={m}>
                {m} months
              </option>
            ))}
          </Form.Control>
        )}
      </div>
      <div className="sub-heading">{k.sub}</div>
      {loading && !res ? (
        <div className="p-3">Looking…</div>
      ) : (
        <>
          <div className="stasharr-cleanup-total">
            <strong>{res?.count ?? 0}</strong> scene(s), <strong>{formatBytes(res?.total_size ?? 0)}</strong>
            {(res?.count ?? 0) > items.length && ` (the biggest ${items.length} are shown)`}
            <span className="flex-grow-1" />
            {items.length > 0 && (
              <Button size="sm" variant="secondary" onClick={() => setSelected(new Set(items.map((i) => i.scene.id)))}>
                Select all shown
              </Button>
            )}
            {selected.size > 0 && (
              <Button size="sm" variant="danger" onClick={remove}>
                Remove {selected.size} ({formatBytes(selectedSize)})
              </Button>
            )}
          </div>
          <Table className="stasharr-table" responsive size="sm">
            <tbody>
              {items.map((i) => (
                <tr key={i.scene.id} onClick={() => toggle(i.scene.id)} className={selected.has(i.scene.id) ? "table-active" : ""}>
                  <td style={{ width: 30 }}>
                    <Form.Check checked={selected.has(i.scene.id)} onChange={() => toggle(i.scene.id)} />
                  </td>
                  <td style={{ width: 110 }}>
                    {i.scene.paths.screenshot && <img src={i.scene.paths.screenshot} alt="" className="stasharr-cleanup-thumb" loading="lazy" />}
                  </td>
                  <td>
                    <Link to={`/scenes/${i.scene.id}`} onClick={(e) => e.stopPropagation()}>
                      {i.scene.title || i.scene.files[0]?.path.split(/[\\/]/).pop()}
                    </Link>
                    <div className="small text-muted">
                      {i.plays === 0 ? "never played" : `${i.plays} play(s), last ${formatDate(i.last_played)}`}
                      {i.rating != null && ` · rating ${Math.round(i.rating / 20)}/5`}
                    </div>
                  </td>
                  <td className="text-right text-nowrap">{formatBytes(i.size)}</td>
                </tr>
              ))}
            </tbody>
          </Table>
          {items.length === 0 && <div className="sub-heading">Nothing to suggest here.</div>}
        </>
      )}
    </div>
  );
};
