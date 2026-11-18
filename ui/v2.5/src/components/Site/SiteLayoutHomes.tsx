// The home pages of the Bento, Mosaic, Spotlight, Showcase and Immersive
// layouts. Classic is in SiteHome.

import React, { useEffect, useRef, useState } from "react";
import { Link, useHistory } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { SiteScene, formatTime, isNew, qualityBadge, sceneTitle } from "./SiteCards";
import { useSite } from "./SiteContext";
import { count, t, views } from "./i18n";

// The banner pieces are shared with Classic.
export interface IHeroParts {
  Slide: React.FC<{ scene: SiteScene; active: boolean }>;
  Text: React.FC<{ scene: SiteScene }>;
}

const femaleOnly = (cfg: GQL.SiteConfigDataFragment) => cfg.models_gender === "FEMALE";

function womenFirst(s: SiteScene) {
  const women = s.performers.filter((p) => p.gender !== GQL.GenderEnum.Male);
  return women.length ? women : s.performers;
}

function starring(s: SiteScene, max = 2): string {
  return womenFirst(s)
    .slice(0, max)
    .map((p) => p.name)
    .join(" & ");
}

// Moves to the next banner every 8 seconds unless paused.
function useRotation(total: number) {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  useEffect(() => {
    if (total < 2 || paused) return;
    const timer = window.setTimeout(() => setIndex((i) => (i + 1) % total), 8000);
    return () => window.clearTimeout(timer);
  }, [index, total, paused]);
  return { index: total ? Math.min(index, total - 1) : 0, setIndex, paused, setPaused };
}

// ---------------------------------------------------------------- Bento

// ---------------------------------------------------------------- Bento

const BentoScene: React.FC<{ scene?: SiteScene; label: string; className: string }> = ({
  scene,
  label,
  className,
}) => {
  if (!scene) return <div className={`site-bento-tile ${className}`} />;
  return (
    <Link to={`/site/scene/${scene.id}`} className={`site-bento-tile has-picture ${className}`}>
      {scene.paths.screenshot && <img src={scene.paths.screenshot} alt="" loading="lazy" />}
      <span className="site-bento-label">{label}</span>
      <h3>{sceneTitle(scene)}</h3>
      <small>{[starring(scene), formatTime(scene.files[0]?.duration)].filter(Boolean).join(" · ")}</small>
    </Link>
  );
};

export const BentoGrid: React.FC<{ scenes: SiteScene[]; parts: IHeroParts }> = ({ scenes, parts }) => {
  const { config } = useSite();
  const { index, setPaused } = useRotation(scenes.length);
  const trending = GQL.useSiteSceneListQuery({ variables: { kind: GQL.SiteSceneKind.Trending, per_page: 1 } });
  const top = GQL.useSiteSceneListQuery({ variables: { kind: GQL.SiteSceneKind.TopRated, per_page: 3 } });
  const latest = GQL.useSiteScenesQuery({
    variables: { filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: 3 } },
    fetchPolicy: "cache-and-network",
  });
  const star = GQL.useSiteModelsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 1 },
      performer_filter: femaleOnly(config)
        ? { gender: { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals } }
        : undefined,
    },
  });
  const tags = GQL.useSiteTagsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 9 },
      tag_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });

  const trend = trending.data?.siteScenes.scenes[0];
  // the top rated pick should not repeat the trending one
  const best = (top.data?.siteScenes.scenes ?? []).find((s) => s.id !== trend?.id);
  const model = star.data?.findPerformers.performers[0];
  const current = scenes[index];

  return (
    <div className="site-bento">
      <div
        className="site-bento-tile site-bento-feature has-picture"
        onMouseEnter={() => setPaused(true)}
        onMouseLeave={() => setPaused(false)}
      >
        {scenes.map((s, i) => (
          <parts.Slide key={s.id} scene={s} active={i === index} />
        ))}
        {current && <parts.Text scene={current} />}
      </div>
      <BentoScene scene={trend} label={t("🔥 Trending #1")} className="site-bento-a" />
      <BentoScene scene={best} label={t("Top rated")} className="site-bento-b" />
      {model ? (
        <Link to={`/site/model/${model.id}`} className="site-bento-tile site-bento-model has-picture">
          {model.image_path && <img src={model.image_path} alt="" loading="lazy" />}
          <span className="site-bento-label">{t("Star of the week")}</span>
          <div className="site-bento-rank">#1</div>
          <h3>{model.name}</h3>
          <small>{count(model.scene_count, "scene")}</small>
        </Link>
      ) : (
        <div className="site-bento-tile site-bento-model" />
      )}
      <div className="site-bento-tile site-bento-cats">
        <h4>{t("Browse")}</h4>
        {(tags.data?.findTags.tags ?? []).map((tag) => (
          <Link key={tag.id} to={`/site/category/${tag.id}`}>
            {tag.name}
          </Link>
        ))}
      </div>
      <div className="site-bento-tile site-bento-new">
        <h4>{t("Just added")}</h4>
        <div className="site-bento-new-list">
          {(latest.data?.findScenes.scenes ?? []).map((s) => (
            <Link key={s.id} to={`/site/scene/${s.id}`}>
              {s.paths.screenshot && <img src={s.paths.screenshot} alt="" loading="lazy" />}
              <span>{sceneTitle(s)}</span>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- Spotlight

type SpotTab = "trending" | "new" | "top" | "popular" | "foryou";

const spotTabs: [SpotTab, string][] = [
  ["trending", "Trending"],
  ["new", "New"],
  ["top", "Top rated"],
  ["popular", "Most viewed"],
  ["foryou", "Picked for you"],
];

function useSpotScenes(tab: SpotTab): SiteScene[] {
  const kind =
    tab === "top" ? GQL.SiteSceneKind.TopRated : tab === "popular" ? GQL.SiteSceneKind.Popular : GQL.SiteSceneKind.Trending;
  const list = GQL.useSiteSceneListQuery({
    skip: tab === "new" || tab === "foryou",
    variables: { kind, per_page: 10 },
  });
  const latest = GQL.useSiteScenesQuery({
    skip: tab !== "new",
    variables: { filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: 10 } },
  });
  const extras = GQL.useSiteHomeExtrasQuery({ skip: tab !== "foryou", fetchPolicy: "cache-and-network" });
  if (tab === "new") return latest.data?.findScenes.scenes ?? [];
  if (tab === "foryou") return (extras.data?.recommendedScenes ?? []).slice(0, 10);
  return list.data?.siteScenes.scenes ?? [];
}

export const SpotlightHome: React.FC<{ fallback: SiteScene[]; parts: IHeroParts }> = ({ fallback, parts }) => {
  const history = useHistory();
  const [tab, setTab] = useState<SpotTab>("trending");
  const [picked, setPicked] = useState(0);
  const hover = useRef<number>();
  const fetched = useSpotScenes(tab);
  // with nothing trending yet (a new library), show the featured scenes
  const scenes = fetched.length ? fetched : fallback;
  useEffect(() => setPicked(0), [tab]);

  if (scenes.length === 0) return null;
  const current = scenes[Math.min(picked, scenes.length - 1)];

  return (
    <div className="site-spot">
      <div className="site-spot-stage">
        <parts.Slide key={current.id} scene={current} active />
        <div className="site-hero-shade" />
        <div className="site-spot-count">
          <b>{String(Math.min(picked, scenes.length - 1) + 1).padStart(2, "0")}</b> / {scenes.length}
        </div>
        <parts.Text scene={current} />
      </div>
      <div className="site-spot-list">
        <div className="site-spot-tabs" role="tablist">
          {spotTabs.map(([id, label]) => (
            <button
              type="button"
              role="tab"
              key={id}
              aria-selected={tab === id}
              className={tab === id ? "active" : ""}
              onClick={() => setTab(id)}
            >
              {t(label)}
            </button>
          ))}
        </div>
        {scenes.map((s, i) => (
          <button
            type="button"
            key={s.id}
            className={`site-spot-row ${s.id === current.id ? "active" : ""}`}
            // pointing at a row for a moment shows it big; a click on the
            // shown one opens it
            onMouseEnter={() => {
              window.clearTimeout(hover.current);
              hover.current = window.setTimeout(() => setPicked(i), 250);
            }}
            onMouseLeave={() => window.clearTimeout(hover.current)}
            onClick={() => (s.id === current.id ? history.push(`/site/scene/${s.id}`) : setPicked(i))}
          >
            <span className="site-spot-num">{String(i + 1).padStart(2, "0")}</span>
            {s.paths.screenshot ? <img src={s.paths.screenshot} alt="" loading="lazy" /> : <span />}
            <div>
              <b>{sceneTitle(s)}</b>
              {starring(s) && <div className="site-scene-models">{starring(s, 3)}</div>}
              <small>
                {[s.studio?.name, formatTime(s.files[0]?.duration), s.site_views > 0 ? views(s.site_views) : ""]
                  .filter(Boolean)
                  .join(" · ")}
              </small>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- Immersive

export const ImmersiveHero: React.FC<{ scenes: SiteScene[]; parts: IHeroParts }> = ({ scenes, parts }) => {
  const { index, setIndex, setPaused } = useRotation(scenes.length);
  if (scenes.length === 0) return null;
  return (
    <section
      className="site-immersive-hero"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      {scenes.map((s, i) => (
        <parts.Slide key={s.id} scene={s} active={i === index} />
      ))}
      <div className="site-hero-shade" />
      <parts.Text scene={scenes[index]} />
      {scenes.length > 1 && (
        <div className="site-hero-dots">
          {scenes.map((s, i) => (
            <button
              type="button"
              key={s.id}
              className={i === index ? "active" : ""}
              onClick={() => setIndex(i)}
              aria-label={t("Show {title}", { title: sceneTitle(s) })}
            />
          ))}
        </div>
      )}
    </section>
  );
};

// ---------------------------------------------------------------- Mosaic

const MosaicCard: React.FC<{ scene: SiteScene; shape: number }> = ({ scene, shape }) => {
  const [hover, setHover] = useState(false);
  const [videoOk, setVideoOk] = useState(true);
  const badge = qualityBadge(scene);
  return (
    <Link
      to={`/site/scene/${scene.id}`}
      className={`site-mosaic-card ar-${shape}`}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      {scene.paths.screenshot ? <img src={scene.paths.screenshot} alt="" loading="lazy" /> : <img alt="" />}
      {hover && scene.paths.preview && videoOk && (
        <video src={scene.paths.preview} autoPlay muted loop playsInline onError={() => setVideoOk(false)} />
      )}
      <div className="site-thumb-badges">
        {isNew(scene) && <span className="site-badge site-badge-new">{t("New")}</span>}
        {badge && <span className="site-badge">{badge}</span>}
      </div>
      <div className="site-mosaic-info">
        <b>{sceneTitle(scene)}</b>
        <small>
          {[starring(scene), formatTime(scene.files[0]?.duration), scene.site_views > 0 ? views(scene.site_views) : ""]
            .filter(Boolean)
            .join(" · ")}
        </small>
      </div>
    </Link>
  );
};

// the card shapes repeat in this order, so the wall looks hand-laid
const shapes = [0, 1, 2, 3, 1, 0, 2];

export const MosaicHome: React.FC<{ promo: React.ReactNode }> = ({ promo }) => {
  const { config } = useSite();
  const female = femaleOnly(config);
  const [tag, setTag] = useState<string | null>(null);
  const [size, setSize] = useState(30);
  useEffect(() => setSize(30), [tag]);
  const tags = GQL.useSiteTagsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 16 },
      tag_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });
  const scenes = GQL.useSiteScenesQuery({
    variables: {
      filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: size },
      scene_filter: tag ? { tags: { value: [tag], modifier: GQL.CriterionModifier.Includes, depth: 0 } } : undefined,
    },
    fetchPolicy: "cache-and-network",
  });
  const models = GQL.useSiteModelsQuery({
    skip: !!tag,
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 4 },
      performer_filter: female
        ? { gender: { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals } }
        : undefined,
    },
  });

  const list = scenes.data?.findScenes.scenes ?? [];
  const total = scenes.data?.findScenes.count ?? 0;
  const stars = tag ? [] : models.data?.findPerformers.performers ?? [];

  // a model card after the 3rd scene, then after every 9th
  const cards: React.ReactNode[] = [];
  let star = 0;
  list.forEach((s, i) => {
    if ((i === 3 || (i > 3 && (i - 3) % 9 === 0)) && star < stars.length) {
      const m = stars[star++];
      cards.push(
        <Link key={`m-${m.id}`} to={`/site/model/${m.id}`} className="site-mosaic-card site-mosaic-model ar-model">
          {m.image_path ? <img src={m.image_path} alt="" loading="lazy" /> : <img alt="" />}
          <span className="site-bento-label">{t(female ? "Pornstar" : "Model")}</span>
          <div className="site-mosaic-info">
            <b>{m.name}</b>
            <small>{count(m.scene_count, "scene")}</small>
          </div>
        </Link>
      );
    }
    cards.push(<MosaicCard key={s.id} scene={s} shape={shapes[i % shapes.length]} />);
  });

  return (
    <div className="site-mosaic">
      <div className="site-mosaic-head">
        <h1>{t("Discover")}</h1>
        {total > 0 && <span>{count(total, "scene")}</span>}
      </div>
      <div className="site-mosaic-chips">
        <button type="button" className={tag === null ? "active" : ""} onClick={() => setTag(null)}>
          {t("All")}
        </button>
        {(tags.data?.findTags.tags ?? []).map((x) => (
          <button type="button" key={x.id} className={tag === x.id ? "active" : ""} onClick={() => setTag(x.id)}>
            {x.name}
          </button>
        ))}
      </div>
      {promo}
      {list.length === 0 && !scenes.loading ? (
        <div className="site-empty">{t("Nothing here yet.")}</div>
      ) : (
        <div className="site-mosaic-wall">{cards}</div>
      )}
      {list.length < total && (
        <button
          type="button"
          className="site-btn site-btn-ghost site-mosaic-more"
          disabled={scenes.loading}
          onClick={() => setSize(size + 30)}
        >
          {t("Load more")}
        </button>
      )}
    </div>
  );
};

// ---------------------------------------------------------------- Showcase

export const ShowcaseHero: React.FC<{ scenes: SiteScene[]; parts: IHeroParts }> = ({ scenes, parts }) => {
  const history = useHistory();
  const n = scenes.length;
  const { index, setIndex, setPaused } = useRotation(n);
  const touch = useRef<number | null>(null);
  const [videoOk, setVideoOk] = useState(true);
  if (n === 0) return null;
  const current = scenes[index];
  const go = (d: number) => setIndex((index + d + n) % n);
  const half = Math.floor(n / 2);

  return (
    <section
      className="site-show"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onTouchStart={(e) => {
        touch.current = e.touches[0].clientX;
      }}
      onTouchEnd={(e) => {
        if (touch.current === null) return;
        const dx = e.changedTouches[0].clientX - touch.current;
        touch.current = null;
        if (Math.abs(dx) > 40) go(dx < 0 ? 1 : -1);
      }}
    >
      <div
        className="site-show-glow"
        style={current.paths.screenshot ? { backgroundImage: `url("${current.paths.screenshot}")` } : undefined}
      />
      <div className="site-show-track">
        {scenes.map((s, i) => {
          // how far this card is from the middle one, going round
          const o = ((i - index + n + half) % n) - half;
          const a = Math.abs(o);
          return (
            <button
              type="button"
              key={s.id}
              className={`site-show-card ${o === 0 ? "center" : ""}`}
              style={{
                transform: `translateX(${o * 46}%) translateZ(${-a * 260}px) rotateY(${o * 38}deg) scale(${1 - a * 0.08})`,
                opacity: a > 2 ? 0 : 1,
                pointerEvents: a > 2 ? "none" : undefined,
                zIndex: 10 - a,
              }}
              aria-label={sceneTitle(s)}
              onClick={() => (o === 0 ? history.push(`/site/scene/${s.id}`) : setIndex(i))}
            >
              {s.paths.screenshot && <img src={s.paths.screenshot} alt="" />}
              {o === 0 && s.paths.preview && videoOk && (
                <video
                  key={s.id}
                  src={s.paths.preview}
                  autoPlay
                  muted
                  loop
                  playsInline
                  onError={() => setVideoOk(false)}
                />
              )}
            </button>
          );
        })}
        {n > 1 && (
          <>
            <button type="button" className="site-show-arrow prev" onClick={() => go(-1)} aria-label={t("Previous")}>
              ‹
            </button>
            <button type="button" className="site-show-arrow next" onClick={() => go(1)} aria-label={t("Next")}>
              ›
            </button>
          </>
        )}
      </div>
      <div className="site-show-info">
        <parts.Text scene={current} />
      </div>
      {n > 1 && (
        <div className="site-show-dots">
          {scenes.map((s, i) => (
            <button
              type="button"
              key={s.id}
              className={i === index ? "active" : ""}
              onClick={() => setIndex(i)}
              aria-label={t("Show {title}", { title: sceneTitle(s) })}
            />
          ))}
        </div>
      )}
    </section>
  );
};
