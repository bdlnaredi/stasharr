import React, { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { ago, t } from "./i18n";
import { sceneTitle } from "./SiteCards";

// The bell in the header: new scenes with your favourite pornstars since you
// last looked. Opening it marks them seen; the list stays until it closes.
export const SiteAlerts: React.FC = () => {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  const { data, refetch } = GQL.useFavoriteAlertsQuery({
    variables: { limit: 12 },
    pollInterval: 5 * 60 * 1000,
    fetchPolicy: "cache-and-network",
  });
  const [markSeen] = GQL.useMarkFavoriteAlertsSeenMutation();
  // what was new when the bell was opened, kept while it is open
  const [shown, setShown] = useState<GQL.FavoriteAlertsQuery["favoriteAlerts"]["alerts"]>([]);

  const count = data?.favoriteAlerts.count ?? 0;

  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, [open]);

  async function toggle() {
    if (open) {
      setOpen(false);
      return;
    }
    setShown(data?.favoriteAlerts.alerts ?? []);
    setOpen(true);
    if (count > 0) {
      await markSeen().catch(() => {});
      refetch();
    }
  }

  return (
    <div className="site-alerts" ref={box}>
      <button type="button" className="site-alerts-bell" onClick={toggle} aria-label={t("Alerts")} title={t("Alerts")}>
        <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true">
          <path
            fill="currentColor"
            d="M12 22a2.5 2.5 0 0 0 2.45-2h-4.9A2.5 2.5 0 0 0 12 22zm7-6V11a7 7 0 0 0-5.5-6.84V3.5a1.5 1.5 0 0 0-3 0v.66A7 7 0 0 0 5 11v5l-2 2v1h18v-1l-2-2z"
          />
        </svg>
        {count > 0 && <span className="site-alerts-badge">{count > 9 ? "9+" : count}</span>}
      </button>
      {open && (
        <div className="site-alerts-menu">
          <div className="site-alerts-head">{t("New from your favourites")}</div>
          {shown.length === 0 ? (
            <div className="site-alerts-empty">
              {t("Nothing new yet. Tap “Add to favourites” on a pornstar's page to hear about her new scenes.")}
            </div>
          ) : (
            shown.map((a) => (
              <Link
                key={a.scene.id}
                to={`/site/scene/${a.scene.id}`}
                className="site-alerts-item"
                onClick={() => setOpen(false)}
              >
                {a.scene.paths.screenshot ? <img src={a.scene.paths.screenshot} alt="" loading="lazy" /> : <span className="ph" />}
                <span className="txt">
                  <b>{sceneTitle(a.scene)}</b>
                  <small>
                    {t("with {name}", { name: a.performer })} · {ago(a.added_at)}
                  </small>
                </span>
              </Link>
            ))
          )}
          <Link to="/site/my-list" className="site-alerts-all" onClick={() => setOpen(false)}>
            {t("My list")} ›
          </Link>
        </div>
      )}
    </div>
  );
};
