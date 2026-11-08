import React, { useState } from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useCurrentUser } from "./CurrentUser";
import { baseName } from "./util";

// Stasharr's part of the Statistics page: what is watched, by whom, and
// how the library and its generated files grow. Admins only.

const PERIODS = [7, 30, 90, 365];

function bytes(n: number): string {
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(n >= 100 || i === 0 ? 0 : 1)} ${units[i]}`;
}

function hours(sec: number): string {
  const h = sec / 3600;
  if (h >= 48) return `${Math.round(h / 24).toLocaleString()} days`;
  return `${h.toFixed(h >= 10 ? 0 : 1)} hours`;
}

function ago(when?: string | null): string {
  if (!when) return "never";
  const secs = (new Date(when).getTime() - Date.now()) / 1000;
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const steps: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31557600],
    ["month", 2629800],
    ["week", 604800],
    ["day", 86400],
    ["hour", 3600],
    ["minute", 60],
  ];
  for (const [unit, size] of steps) {
    if (Math.abs(secs) >= size) return rtf.format(Math.round(secs / size), unit);
  }
  return "just now";
}

function title(s: { title?: string | null; files: { path: string }[] }): string {
  return s.title || baseName(s.files[0]?.path ?? "") || "Untitled";
}

const Card: React.FC<{ value: React.ReactNode; label: string }> = ({ value, label }) => (
  <div className="stasharr-stat-card">
    <div className="value">{value}</div>
    <div className="label">{label}</div>
  </div>
);

// a simple bar chart; labels under each bar
const Bars: React.FC<{ items: { label: string; value: number; tip?: string }[]; height?: number }> = ({
  items,
  height = 120,
}) => {
  const max = Math.max(1, ...items.map((i) => i.value));
  return (
    <div className="stasharr-bars" style={{ height: height + 34 }}>
      {items.map((i) => (
        <div className="bar" key={i.label} title={i.tip ?? `${i.label}: ${i.value.toLocaleString()}`}>
          <span className="num">{i.value > 0 ? i.value.toLocaleString() : ""}</span>
          <div className="fill" style={{ height: Math.max(2, (i.value / max) * height) }} />
          <span className="lbl">{i.label}</span>
        </div>
      ))}
    </div>
  );
};

// a list of names with a bar for each count
const Ranked: React.FC<{ items: { key: string; name: React.ReactNode; count: number }[]; unit: string }> = ({
  items,
  unit,
}) => {
  const max = Math.max(1, ...items.map((i) => i.count));
  if (items.length === 0) return <p className="text-muted">Nothing yet.</p>;
  return (
    <ol className="stasharr-ranked">
      {items.map((i) => (
        <li key={i.key}>
          <div className="row-text">
            <span className="name">{i.name}</span>
            <span className="count">
              {i.count.toLocaleString()} {unit}
            </span>
          </div>
          <div className="track">
            <div className="fill" style={{ width: `${(i.count / max) * 100}%` }} />
          </div>
        </li>
      ))}
    </ol>
  );
};

export const StasharrStats: React.FC = () => {
  const { isAdmin } = useCurrentUser();
  const [days, setDays] = useState(30);
  const { data, loading, error } = GQL.useStasharrStatsQuery({
    variables: { days },
    skip: !isAdmin,
    fetchPolicy: "cache-and-network",
  });
  if (!isAdmin) return null;
  if (error) return <p className="text-danger text-center">{error.message}</p>;
  if (loading && !data) return <LoadingIndicator />;
  const s = data?.stasharrStats;
  if (!s) return null;
  const t = s.totals;

  // fill in days without plays so the chart has no gaps
  const dayItems: { label: string; value: number; tip: string }[] = [];
  const byDay = new Map(s.plays_by_day.map((d) => [d.day, d.plays]));
  const shown = Math.min(days, 90);
  for (let i = shown - 1; i >= 0; i--) {
    const d = new Date(Date.now() - i * 86400000);
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
    const v = byDay.get(key) ?? 0;
    const label = shown <= 31 ? String(d.getDate()) : i % 7 === 0 ? `${d.getDate()}/${d.getMonth() + 1}` : "";
    dayItems.push({ label: label || " ".repeat((i % 7) + 1), value: v, tip: `${d.toLocaleDateString()}: ${v} plays` });
  }

  return (
    <div className="stasharr-stats">
      <div className="stasharr-stats-head">
        <h2>Watching and library</h2>
        <div className="btn-group">
          {PERIODS.map((p) => (
            <button
              key={p}
              type="button"
              className={`btn btn-sm ${p === days ? "btn-primary" : "btn-secondary"}`}
              onClick={() => setDays(p)}
            >
              {p === 365 ? "Year" : `${p} days`}
            </button>
          ))}
        </div>
      </div>

      <div className="stasharr-stat-cards">
        <Card value={s.plays_by_day.reduce((a, d) => a + d.plays, 0).toLocaleString()} label={`plays in ${days} days`} />
        <Card value={t.scenes.toLocaleString()} label="scenes" />
        <Card value={hours(t.duration)} label="of video" />
        <Card value={bytes(t.size)} label="video files" />
        <Card value={bytes(s.generated_total)} label="generated files" />
        <Card value={t.reels.toLocaleString()} label="reels" />
        <Card value={t.reel_views.toLocaleString()} label="reel views" />
        <Card value={t.comments.toLocaleString()} label="comments" />
        <Card value={t.users.toLocaleString()} label="accounts" />
      </div>

      <section>
        <h3>Plays per day</h3>
        <Bars items={dayItems} />
      </section>

      <div className="stasharr-stats-grid">
        <section>
          <h3>Most watched scenes</h3>
          <Ranked
            unit="plays"
            items={s.top_scenes.map((x) => ({
              key: x.scene.id,
              name: (
                <Link to={`/scenes/${x.scene.id}`} className="with-thumb">
                  {x.scene.paths.screenshot && <img src={x.scene.paths.screenshot} alt="" loading="lazy" />}
                  {title(x.scene)}
                </Link>
              ),
              count: x.plays,
            }))}
          />
        </section>
        <section>
          <h3>Most watched performers</h3>
          <Ranked
            unit="plays"
            items={s.top_performers.map((x) => ({
              key: x.id,
              name: <Link to={`/performers/${x.id}`}>{x.name}</Link>,
              count: x.count,
            }))}
          />
        </section>
        <section>
          <h3>Most watched tags</h3>
          <Ranked
            unit="plays"
            items={s.top_tags.map((x) => ({ key: x.id, name: <Link to={`/tags/${x.id}`}>{x.name}</Link>, count: x.count }))}
          />
        </section>
        <section>
          <h3>Most watched reels</h3>
          <Ranked
            unit="views"
            items={s.top_reels.map((x) => ({
              key: x.reel_id,
              name: <Link to={`/scenes/${x.scene.id}`}>{title(x.scene)}</Link>,
              count: x.views,
            }))}
          />
        </section>
      </div>

      <section>
        <h3>Who is watching</h3>
        <div className="table-responsive">
          <table className="table table-sm stasharr-stats-table">
            <thead>
              <tr>
                <th>Account</th>
                <th>Role</th>
                <th>Plays ({days} days)</th>
                <th>Last played</th>
                <th>Last signed in</th>
              </tr>
            </thead>
            <tbody>
              {s.users.map((u) => (
                <tr key={u.id}>
                  <td>{u.name}</td>
                  <td>{u.role.charAt(0) + u.role.slice(1).toLowerCase()}</td>
                  <td>{u.plays.toLocaleString()}</td>
                  <td>{ago(u.last_play)}</td>
                  <td>{u.id === "0" ? "—" : ago(u.last_login)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h3>Recently played</h3>
        {s.recent_plays.length === 0 ? (
          <p className="text-muted">Nothing yet.</p>
        ) : (
          <ul className="stasharr-recent">
            {s.recent_plays.map((p, i) => (
              <li key={i}>
                <b>{p.user}</b> watched <Link to={`/scenes/${p.scene.id}`}>{title(p.scene)}</Link>{" "}
                <span className="text-muted">{ago(p.at)}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <div className="stasharr-stats-grid">
        <section>
          <h3>Scenes added per month</h3>
          <Bars
            items={s.added_by_month.map((m) => ({
              label: new Date(`${m.month}-01T00:00:00`).toLocaleDateString(undefined, { month: "short" }),
              value: m.scenes,
              tip: `${m.month}: ${m.scenes} scenes, ${bytes(m.size)}`,
            }))}
          />
        </section>
        <section>
          <h3>Picture quality</h3>
          <Ranked unit="scenes" items={s.resolutions.map((r) => ({ key: r.name, name: r.name, count: r.count }))} />
        </section>
        <section>
          <h3>Storage</h3>
          <Ranked
            unit=""
            items={[
              { key: "video", name: `Video files: ${bytes(t.size)}`, count: Math.round(t.size / 1048576) },
              ...s.generated.map((g) => ({
                key: g.name,
                name: `${g.name}: ${bytes(g.bytes)}`,
                count: Math.round(g.bytes / 1048576),
              })),
            ]}
          />
          <p className="text-muted small">Bars are in MB. Generated folder sizes are updated every 10 minutes.</p>
        </section>
      </div>
    </div>
  );
};
