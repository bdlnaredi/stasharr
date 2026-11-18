// The monthly recap: a private page with the signed-in profile's month.

import React, { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { Loading, SceneTile } from "./SiteCards";
import { getLang, t } from "./i18n";

function monthName(m: string): string {
  const [y, mo] = m.split("-").map(Number);
  if (!y || !mo) return m;
  return new Date(y, mo - 1, 1).toLocaleDateString(getLang(), { month: "long", year: "numeric" });
}

function watchTime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.round((seconds % 3600) / 60);
  if (h === 0) return t("{m} min", { m: String(m) });
  return t("{h} h {m} min", { h: String(h), m: String(m) });
}

// What kind of watcher the hours say you are.
function watcherType(hours: number[]): { name: string; line: string } | null {
  const total = hours.reduce((a, b) => a + b, 0);
  if (total === 0) return null;
  const sum = (from: number, to: number) => {
    let n = 0;
    for (let h = from; h !== to; h = (h + 1) % 24) n += hours[h];
    return n;
  };
  const parts = [
    { name: t("Night owl"), line: t("Most of your plays were after 10 at night."), n: sum(22, 4) },
    { name: t("Early bird"), line: t("You liked to start the day with a scene."), n: sum(5, 10) },
    { name: t("Lunch-break viewer"), line: t("Midday was your favourite time."), n: sum(11, 15) },
    { name: t("Evening regular"), line: t("Evenings were when you pressed play."), n: sum(17, 22) },
  ];
  const best = parts.sort((a, b) => b.n - a.n)[0];
  return best.n / total >= 0.4 ? best : { name: t("Any-time viewer"), line: t("You watched at all hours.") };
}

const Bars: React.FC<{ values: number[]; labels: string[]; highlight?: boolean }> = ({ values, labels, highlight }) => {
  const max = Math.max(1, ...values);
  const top = values.indexOf(Math.max(...values));
  return (
    <div className="site-recap-bars" role="img" aria-label={labels.map((l, i) => `${l}: ${values[i]}`).join(", ")}>
      {values.map((v, i) => (
        <div key={i} className={`site-recap-bar ${highlight && i === top && v > 0 ? "top" : ""}`}>
          <span style={{ height: `${Math.max(2, (v / max) * 100)}%` }} title={`${labels[i]}: ${v}`} />
          <small>{labels[i]}</small>
        </div>
      ))}
    </div>
  );
};

export const RecapPage: React.FC = () => {
  const months = GQL.useRecapMonthsQuery({ fetchPolicy: "cache-and-network" });
  const list = months.data?.recapMonths ?? [];
  const [month, setMonth] = useState<string | null>(null);
  useEffect(() => {
    if (!month && list.length) setMonth(list[0]);
  }, [list, month]);
  const { data, loading } = GQL.useMonthRecapQuery({
    variables: { month },
    skip: !month,
    fetchPolicy: "cache-and-network",
  });

  if (months.loading && !months.data) return <Loading />;
  if (list.length === 0) {
    return (
      <div className="site-page site-recap">
        <h1 className="site-page-title">{t("Monthly recap")}</h1>
        <p className="site-muted">{t("Watch a few scenes and your first recap shows up here.")}</p>
      </div>
    );
  }

  const r = data?.monthRecap;
  const who = r ? watcherType(r.hours) : null;
  const dayNames = [t("Sun"), t("Mon"), t("Tue"), t("Wed"), t("Thu"), t("Fri"), t("Sat")];
  const hourLabels = Array.from({ length: 24 }, (_, h) => (h % 6 === 0 ? String(h).padStart(2, "0") : ""));
  const star = r?.models[0];

  return (
    <div className="site-page site-recap">
      <div className="site-recap-head">
        <div>
          <span className="site-kicker">{t("Your monthly recap")}</span>
          <h1 className="site-page-title">{month ? monthName(month) : ""}</h1>
          <p className="site-muted">{t("Only you can see this page.")}</p>
        </div>
        <select value={month ?? ""} onChange={(e) => setMonth(e.target.value)} aria-label={t("Month")}>
          {list.map((m) => (
            <option key={m} value={m}>
              {monthName(m)}
            </option>
          ))}
        </select>
      </div>

      {!r && loading && <Loading />}
      {r && (
        <>
          <div className="site-recap-stats">
            <div className="site-recap-stat site-recap-hero">
              <small>{t("Time watched")}</small>
              <b>{r.seconds > 0 ? watchTime(r.seconds) : "—"}</b>
              {r.seconds === 0 && <span>{t("Watch time is counted from this update on.")}</span>}
            </div>
            <div className="site-recap-stat">
              <small>{t("Plays")}</small>
              <b>{r.plays}</b>
            </div>
            <div className="site-recap-stat">
              <small>{t("Different scenes")}</small>
              <b>{r.scenes}</b>
            </div>
            <div className="site-recap-stat">
              <small>{t("Days you watched")}</small>
              <b>{r.active_days}</b>
            </div>
            <div className="site-recap-stat">
              <small>{t("Longest streak")}</small>
              <b>{t("{n} days", { n: String(r.longest_run) })}</b>
            </div>
            <div className="site-recap-stat">
              <small>{t("Scenes you liked")}</small>
              <b>{r.likes}</b>
            </div>
          </div>

          {star && (
            <section className="site-recap-star">
              <Link to={`/site/model/${star.performer.id}`} className="site-recap-star-photo">
                {star.performer.image_path && <img src={star.performer.image_path} alt="" />}
                <span>#1</span>
              </Link>
              <div>
                <span className="site-kicker">{t("Your top star")}</span>
                <h2>
                  <Link to={`/site/model/${star.performer.id}`}>{star.performer.name}</Link>
                </h2>
                <p>{t("{n} plays this month", { n: String(star.plays) })}</p>
                <ol className="site-recap-list" start={2}>
                  {r.models.slice(1).map((m) => (
                    <li key={m.performer.id}>
                      <Link to={`/site/model/${m.performer.id}`}>
                        {m.performer.image_path && <img src={m.performer.image_path} alt="" />}
                        <span>{m.performer.name}</span>
                        <small>{t("{n} plays", { n: String(m.plays) })}</small>
                      </Link>
                    </li>
                  ))}
                </ol>
              </div>
            </section>
          )}

          <div className="site-recap-grid">
            {r.categories.length > 0 && (
              <section className="site-recap-card">
                <h3>{t("Top categories")}</h3>
                {r.categories.map((c) => (
                  <Link key={c.tag.id} to={`/site/category/${c.tag.id}`} className="site-recap-meter">
                    <span>{c.tag.name}</span>
                    <span className="site-recap-meter-bar">
                      <span style={{ width: `${(c.plays / r.categories[0].plays) * 100}%` }} />
                    </span>
                    <small>{c.plays}</small>
                  </Link>
                ))}
              </section>
            )}
            {r.studios.length > 0 && (
              <section className="site-recap-card">
                <h3>{t("Top channels")}</h3>
                {r.studios.map((s, i) => (
                  <Link key={s.studio.id} to={`/site/studio/${s.studio.id}`} className="site-recap-channel">
                    <em>{i + 1}</em>
                    <span>{s.studio.name}</span>
                    <small>{t("{n} plays", { n: String(s.plays) })}</small>
                  </Link>
                ))}
              </section>
            )}
            <section className="site-recap-card">
              <h3>{t("When you watch")}</h3>
              {who && (
                <p className="site-recap-type">
                  <b>{who.name}</b> {who.line}
                </p>
              )}
              <Bars values={r.hours} labels={hourLabels} highlight />
              <Bars values={r.weekdays} labels={dayNames} highlight />
            </section>
            {r.top_scene && (
              <section className="site-recap-card">
                <h3>{t("Most replayed")}</h3>
                <SceneTile scene={r.top_scene} />
                <p className="site-muted">{t("{n} plays", { n: String(r.top_scene_plays) })}</p>
              </section>
            )}
          </div>
        </>
      )}
    </div>
  );
};
