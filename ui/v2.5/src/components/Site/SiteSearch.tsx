import React, { useEffect, useRef, useState } from "react";
import { Link, useHistory, useLocation } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { formatTime, sceneTitle } from "./SiteCards";
import { useSite } from "./SiteContext";
import { count, t } from "./i18n";

// waits this long after the last key press before searching
const DELAY = 250;

function useDebounced(value: string, ms: number): string {
  const [out, setOut] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setOut(value), ms);
    return () => window.clearTimeout(timer);
  }, [value, ms]);
  return out;
}

// The header search box. Results appear in a dropdown while typing; Enter
// or "See all results" opens the full results page.
export const SiteSearch: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const { config } = useSite();
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  // phones show a search button that opens the box
  const [phoneOpen, setPhoneOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const box = useRef<HTMLFormElement>(null);
  const input = useRef<HTMLInputElement>(null);

  const term = useDebounced(q.trim(), DELAY);
  const searching = term.length >= 2;

  const scenes = GQL.useSiteScenesQuery({
    variables: {
      filter: { q: term, per_page: 6, sort: "created_at", direction: GQL.SortDirectionEnum.Desc },
    },
    skip: !searching,
    fetchPolicy: "cache-first",
  });
  const models = GQL.useSiteModelsQuery({
    variables: {
      filter: { q: term, per_page: 5, sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc },
      performer_filter:
        config.models_gender === "FEMALE"
          ? { gender: { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals } }
          : undefined,
    },
    skip: !searching,
    fetchPolicy: "cache-first",
  });

  const sceneList = searching ? scenes.data?.findScenes.scenes ?? [] : [];
  const sceneCount = searching ? scenes.data?.findScenes.count ?? 0 : 0;
  const modelList = searching ? models.data?.findPerformers.performers ?? [] : [];
  const loading = searching && (scenes.loading || models.loading) && !sceneList.length && !modelList.length;

  // every result in order, for the arrow keys
  const links = [
    ...modelList.map((m) => `/site/model/${m.id}`),
    ...sceneList.map((s) => `/site/scene/${s.id}`),
  ];

  useEffect(() => setActive(-1), [term]);

  // close when the page changes
  useEffect(() => {
    setOpen(false);
    setPhoneOpen(false);
  }, [location.pathname]);

  // close when clicking elsewhere
  useEffect(() => {
    function onDown(e: MouseEvent) {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, []);

  useEffect(() => {
    if (phoneOpen) input.current?.focus();
  }, [phoneOpen]);

  function showAll() {
    const text = q.trim();
    if (!text) return;
    setOpen(false);
    input.current?.blur();
    history.push(`/site/search?q=${encodeURIComponent(text)}`);
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (active >= 0 && links[active]) {
      setOpen(false);
      history.push(links[active]);
    } else {
      showAll();
    }
  }

  function onKey(e: React.KeyboardEvent) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setOpen(true);
      setActive((a) => Math.min(links.length - 1, a + 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) => Math.max(-1, a - 1));
    } else if (e.key === "Escape") {
      setOpen(false);
      setPhoneOpen(false);
      input.current?.blur();
    }
  }

  const showDropdown = open && q.trim().length >= 2;
  const nothing = searching && !loading && !sceneList.length && !modelList.length;

  return (
    <>
      <button
        type="button"
        className="site-search-toggle"
        aria-label={t("Search")}
        onClick={() => setPhoneOpen(!phoneOpen)}
      >
        ⌕
      </button>
      <form
        ref={box}
        className={`site-search ${phoneOpen ? "phone-open" : ""}`}
        onSubmit={onSubmit}
        role="search"
      >
        <input
          ref={input}
          type="search"
          placeholder={t("Search scenes and stars")}
          value={q}
          autoComplete="off"
          onChange={(e) => {
            setQ(e.currentTarget.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKey}
        />
        <button type="submit" aria-label={t("Search")}>
          ⌕
        </button>

        {showDropdown && (
          <div className="site-search-results">
            {loading && <div className="site-search-note">{t("Searching…")}</div>}
            {nothing && <div className="site-search-note">{t("Nothing found for “{q}”.", { q: q.trim() })}</div>}

            {modelList.length > 0 && (
              <>
                <div className="site-search-head">{t(config.models_gender === "FEMALE" ? "Pornstars" : "Models")}</div>
                <div className="site-search-models">
                  {modelList.map((m, i) => (
                    <Link
                      key={m.id}
                      to={`/site/model/${m.id}`}
                      className={`site-search-model ${active === i ? "active" : ""}`}
                      onMouseEnter={() => setActive(i)}
                    >
                      <span className="site-search-face">
                        {m.image_path && <img src={m.image_path} alt="" />}
                      </span>
                      <span>{m.name}</span>
                    </Link>
                  ))}
                </div>
              </>
            )}

            {sceneList.length > 0 && (
              <>
                <div className="site-search-head">{t("Scenes")}</div>
                {sceneList.map((s, j) => {
                  const i = modelList.length + j;
                  const who = s.performers
                    .filter((p) => p.gender !== GQL.GenderEnum.Male)
                    .slice(0, 2)
                    .map((p) => p.name)
                    .join(", ");
                  return (
                    <Link
                      key={s.id}
                      to={`/site/scene/${s.id}`}
                      className={`site-search-scene ${active === i ? "active" : ""}`}
                      onMouseEnter={() => setActive(i)}
                    >
                      <span className="site-search-thumb">
                        {s.paths.screenshot && <img src={s.paths.screenshot} alt="" />}
                        {s.files[0]?.duration ? <em>{formatTime(s.files[0].duration)}</em> : null}
                      </span>
                      <span className="site-search-text">
                        <b>{sceneTitle(s)}</b>
                        <small>{[who, s.studio?.name].filter(Boolean).join(" · ")}</small>
                      </span>
                    </Link>
                  );
                })}
              </>
            )}

            {(sceneList.length > 0 || modelList.length > 0) && (
              <button type="button" className="site-search-all" onClick={showAll}>
                {t("See all results")}
                {sceneCount > sceneList.length ? ` (${count(sceneCount, "scene")})` : ""} ›
              </button>
            )}
          </div>
        )}
      </form>
    </>
  );
};
