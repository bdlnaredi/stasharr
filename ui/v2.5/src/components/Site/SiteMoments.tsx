import React from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { Section } from "./SiteCards";
import { sceneTitle } from "./SiteCards";
import { t } from "./i18n";

function clock(sec: number): string {
  const s = Math.max(0, Math.round(sec));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const r = String(s % 60).padStart(2, "0");
  return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${r}` : `${m}:${r}`;
}

// Moments that look like what was typed, found by the Stasharr Tagger.
// Shows nothing when search by description is not set up.
export const MomentsSection: React.FC<{ q: string; limit?: number }> = ({ q, limit = 12 }) => {
  const { data, loading } = GQL.useSearchMomentsQuery({
    variables: { query: q, limit },
    skip: q.trim().length < 3,
    fetchPolicy: "cache-first",
    errorPolicy: "ignore",
  });
  const list = data?.searchMoments ?? [];
  if (loading && list.length === 0) return null;
  if (list.length === 0) return null;
  return (
    <Section title={t("Matching moments")} className="site-moments">
      <div className="site-moment-grid">
        {list.map((m) => (
          <Link key={m.scene.id} to={`/site/scene/${m.scene.id}?t=${Math.floor(m.seconds)}`} className="site-moment">
            <div className="site-moment-pic">
              <img src={m.frame} alt="" loading="lazy" />
              <span className="site-moment-time">▶ {clock(m.seconds)}</span>
            </div>
            <div className="site-moment-title">{sceneTitle(m.scene)}</div>
          </Link>
        ))}
      </div>
    </Section>
  );
};
