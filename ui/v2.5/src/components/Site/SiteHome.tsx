import React, { useEffect, useMemo, useState } from "react";
import { Link, useHistory } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import {
  CategoryTile,
  Loading,
  ModelTile,
  Rail,
  SceneGrid,
  SceneTile,
  Section,
  SiteScene,
  StudioTile,
  formatDate,
  formatTime,
  qualityBadge,
  sceneTitle,
  titleSize,
} from "./SiteCards";
import { ISiteStyle } from "./siteStyles";
import { useSite } from "./SiteContext";
import { ReelsRow } from "./SiteReels";
import { LikeButton } from "./SiteWidgets";
import { t } from "./i18n";

// ---------------------------------------------------------------- hero

const HeroSlide: React.FC<{ scene: SiteScene; active: boolean }> = ({ scene, active }) => {
  const [videoOk, setVideoOk] = useState(true);
  return (
    <div className={`site-hero-slide ${active ? "active" : ""}`}>
      {scene.paths.screenshot && <img src={scene.paths.screenshot} alt="" />}
      {active && scene.paths.preview && videoOk && (
        <video
          src={scene.paths.preview}
          autoPlay
          muted
          loop
          playsInline
          onError={() => setVideoOk(false)}
        />
      )}
    </div>
  );
};

const HeroText: React.FC<{ scene: SiteScene }> = ({ scene }) => {
  const history = useHistory();
  const [inList, setInList] = useState(scene.in_watch_later);
  const [toggle] = GQL.useWatchLaterToggleMutation();
  useEffect(() => setInList(scene.in_watch_later), [scene]);
  const models = scene.performers.filter((p) => p.gender !== GQL.GenderEnum.Male);
  const badge = qualityBadge(scene);

  return (
    <div className="site-hero-text" key={scene.id}>
      <div className="site-hero-kicker">
        <span>{t("Featured")}</span>
        {scene.studio && <Link to={`/site/studio/${scene.studio.id}`}>{scene.studio.name}</Link>}
      </div>
      <h1 className={titleSize(sceneTitle(scene))} title={sceneTitle(scene)}>
        {sceneTitle(scene)}
      </h1>
      {(models.length > 0 || scene.performers.length > 0) && (
        <div className="site-hero-models">
          {t("Starring")}{" "}
          {(models.length ? models : scene.performers).slice(0, 4).map((p, i) => (
            <React.Fragment key={p.id}>
              {i > 0 && " & "}
              <Link to={`/site/model/${p.id}`}>{p.name}</Link>
            </React.Fragment>
          ))}
        </div>
      )}
      <div className="site-hero-meta">
        {scene.date && <span>{formatDate(scene.date)}</span>}
        {scene.files[0]?.duration ? <span>{formatTime(scene.files[0].duration)}</span> : null}
        {badge && <span className="site-badge">{badge}</span>}
      </div>
      {scene.details && <p className="site-hero-details">{scene.details}</p>}
      <div className="site-hero-actions">
        <button type="button" className="site-btn site-btn-primary" onClick={() => history.push(`/site/scene/${scene.id}`)}>
          {t("▶ Watch now")}
        </button>
        <button
          type="button"
          className="site-btn site-btn-ghost"
          onClick={async () => {
            try {
              const res = await toggle({ variables: { scene_id: scene.id } });
              setInList(!!res.data?.watchLaterToggle);
            } catch {
              // ignore
            }
          }}
        >
          {inList ? t("✓ In my list") : t("+ My list")}
        </button>
        <LikeButton sceneID={scene.id} label={false} />
      </div>
    </div>
  );
};

const Hero: React.FC<{ scenes: SiteScene[]; style: ISiteStyle }> = ({ scenes, style }) => {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const count = scenes.length;

  useEffect(() => {
    if (count < 2 || paused) return;
    const timer = window.setTimeout(() => setIndex((i) => (i + 1) % count), 8000);
    return () => window.clearTimeout(timer);
  }, [index, count, paused]);

  if (count === 0) return null;
  const current = scenes[Math.min(index, count - 1)];
  const next = [1, 2, 3].map((d) => scenes[(index + d) % count]).filter((s, i, a) => count > 1 && s.id !== current.id && a.indexOf(s) === i);

  return (
    <section
      className={`site-hero site-hero-${style.hero}`}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="site-hero-stage">
        {scenes.map((s, i) => (
          <HeroSlide key={s.id} scene={s} active={i === index} />
        ))}
        <div className="site-hero-shade" />
        <HeroText scene={current} />
        {count > 1 && (
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
      </div>
      {style.hero === "split" && next.length > 0 && (
        <div className="site-hero-side">
          <div className="site-hero-side-title">{t("Up next")}</div>
          {next.map((s) => (
            <button
              type="button"
              key={s.id}
              className="site-hero-side-item"
              onClick={() => setIndex(scenes.indexOf(s))}
            >
              {s.paths.screenshot && <img src={s.paths.screenshot} alt="" />}
              <span>{sceneTitle(s)}</span>
            </button>
          ))}
        </div>
      )}
    </section>
  );
};

// ---------------------------------------------------------------- ticker

export const Ticker: React.FC = () => {
  const { data } = GQL.useSiteScenesQuery({
    variables: { filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: 15 } },
    fetchPolicy: "cache-first",
  });
  const scenes = data?.findScenes.scenes ?? [];
  if (scenes.length === 0) return null;
  const items = scenes.map((s) => {
    const who = s.performers.filter((p) => p.gender !== GQL.GenderEnum.Male).map((p) => p.name);
    return { id: s.id, text: sceneTitle(s), who: who.slice(0, 2).join(" & "), date: s.created_at };
  });
  // twice, so the loop has no gap
  const loop = [...items, ...items];
  return (
    <div className="site-ticker">
      <span className="site-ticker-label">{t("Just added")}</span>
      <div className="site-ticker-window">
        <div className="site-ticker-track" style={{ animationDuration: `${items.length * 6}s` }}>
          {loop.map((it, i) => (
            <Link key={`${it.id}-${i}`} to={`/site/scene/${it.id}`} className="site-ticker-item">
              <span className="site-ticker-dot">●</span>
              {it.text}
              {it.who && <em> — {it.who}</em>}
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- rows

const female = (cfg: GQL.SiteConfigDataFragment) => cfg.models_gender === "FEMALE";

const SceneRail: React.FC<{ title: string; link?: string; scenes: SiteScene[] }> = ({
  title,
  link,
  scenes,
}) =>
  scenes.length === 0 ? null : (
    <Section title={title} link={link}>
      <Rail>
        {scenes.map((s) => (
          <SceneTile key={s.id} scene={s} />
        ))}
      </Rail>
    </Section>
  );

const ListRow: React.FC<{
  kind: GQL.SiteSceneKind;
  title: string;
  link: string;
  grid?: "normal" | "large";
}> = ({ kind, title, link, grid }) => {
  const { data } = GQL.useSiteSceneListQuery({
    variables: { kind, per_page: grid === "large" ? 4 : 12 },
  });
  const scenes = data?.siteScenes.scenes ?? [];
  if (scenes.length === 0) return null;
  if (grid) {
    return (
      <Section title={title} link={link}>
        <SceneGrid scenes={scenes} large={grid === "large"} />
      </Section>
    );
  }
  return <SceneRail title={title} link={link} scenes={scenes} />;
};

const LatestRow: React.FC<{ title: string }> = ({ title }) => {
  const { data } = GQL.useSiteScenesQuery({
    variables: { filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: 8 } },
    fetchPolicy: "cache-and-network",
  });
  const scenes = data?.findScenes.scenes ?? [];
  if (scenes.length === 0) return null;
  return (
    <Section title={title} link="/site/scenes">
      <SceneGrid scenes={scenes} />
    </Section>
  );
};

const TagRow: React.FC<{ tag: { id: string; name: string }; title: string }> = ({ tag, title }) => {
  const { data } = GQL.useSiteScenesQuery({
    variables: {
      filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: 12 },
      scene_filter: { tags: { value: [tag.id], modifier: GQL.CriterionModifier.Includes, depth: 0 } },
    },
  });
  return <SceneRail title={title || tag.name} link={`/site/category/${tag.id}`} scenes={data?.findScenes.scenes ?? []} />;
};

const ModelsRow: React.FC<{ title: string }> = ({ title }) => {
  const { config } = useSite();
  const { data } = GQL.useSiteModelsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 16 },
      performer_filter: female(config)
        ? { gender: { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals } }
        : undefined,
    },
  });
  const models = data?.findPerformers.performers ?? [];
  if (models.length === 0) return null;
  return (
    <Section title={title} link="/site/models" className="site-section-models">
      <Rail>
        {models.map((m, i) => (
          <ModelTile key={m.id} model={m} rank={i + 1} />
        ))}
      </Rail>
    </Section>
  );
};

const FavoritesRow: React.FC<{ title: string }> = ({ title }) => {
  const { data } = GQL.useSiteFromFavoritesQuery({
    variables: { per_page: 12 },
    fetchPolicy: "cache-and-network",
  });
  return <SceneRail title={title} link="/site/my-list" scenes={data?.siteFromFavorites.scenes ?? []} />;
};

const LikedRow: React.FC<{ title: string }> = ({ title }) => {
  const { data } = GQL.useSiteLikedQuery({ variables: { per_page: 12 }, fetchPolicy: "cache-and-network" });
  return <SceneRail title={title} link="/site/my-list" scenes={data?.siteLikedScenes.scenes ?? []} />;
};

const ExtrasRow: React.FC<{ which: "continue" | "recommended"; title: string }> = ({ which, title }) => {
  const { data } = GQL.useSiteHomeExtrasQuery({ fetchPolicy: "cache-and-network" });
  const scenes = (which === "continue" ? data?.continueWatching : data?.recommendedScenes) ?? [];
  return <SceneRail title={title} link={which === "continue" ? "/site/my-list" : undefined} scenes={scenes} />;
};

const CategoriesRow: React.FC<{ title: string }> = ({ title }) => {
  const { data } = GQL.useSiteTagsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 12 },
      tag_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });
  const tags = data?.findTags.tags ?? [];
  if (tags.length === 0) return null;
  return (
    <Section title={title} link="/site/categories">
      <div className="site-category-grid">
        {tags.map((tag) => (
          <CategoryTile key={tag.id} tag={tag} />
        ))}
      </div>
    </Section>
  );
};

const ChannelsRow: React.FC<{ title: string }> = ({ title }) => {
  const { data } = GQL.useSiteStudiosQuery({
    variables: { filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 12 } },
  });
  const studios = (data?.findStudios.studios ?? []).filter((s) => s.scene_count > 0);
  if (studios.length === 0) return null;
  return (
    <Section title={title} link="/site/channels">
      <Rail>
        {studios.map((s) => (
          <StudioTile key={s.id} studio={s} />
        ))}
      </Rail>
    </Section>
  );
};

// the heading each row type gets when the admin gave none
export function defaultRowTitle(type: string, femaleOnly: boolean): string {
  switch (type) {
    case "continue":
      return "Continue watching";
    case "latest":
      return "Latest updates";
    case "models":
      return femaleOnly ? "Top pornstars" : "Top models";
    case "trending":
      return "Trending now";
    case "popular":
      return "Most viewed";
    case "recommended":
      return "Picked for you";
    case "favorites":
      return "New from your favourites";
    case "liked":
      return "Your liked scenes";
    case "reels":
      return "Reels";
    case "top_rated":
      return "Top rated";
    case "categories":
      return "Categories";
    case "channels":
      return "Channels";
    default:
      return "";
  }
}

const HomeRow: React.FC<{ row: GQL.SiteConfigDataFragment["home_rows"][number] }> = ({ row }) => {
  const { config } = useSite();
  const title = row.title || t(defaultRowTitle(row.type, female(config)));
  switch (row.type) {
    case "continue":
      return <ExtrasRow which="continue" title={title} />;
    case "recommended":
      return <ExtrasRow which="recommended" title={title} />;
    case "latest":
      return <LatestRow title={title} />;
    case "models":
      return <ModelsRow title={title} />;
    case "trending":
      return <ListRow kind={GQL.SiteSceneKind.Trending} title={title} link="/site/scenes?sort=trending" />;
    case "popular":
      return <ListRow kind={GQL.SiteSceneKind.Popular} title={title} link="/site/scenes?sort=popular" />;
    case "top_rated":
      return <ListRow kind={GQL.SiteSceneKind.TopRated} title={title} link="/site/scenes?sort=top" grid="large" />;
    case "favorites":
      return <FavoritesRow title={title} />;
    case "liked":
      return <LikedRow title={title} />;
    case "reels":
      return <ReelsRow title={title} />;
    case "categories":
      return <CategoriesRow title={title} />;
    case "channels":
      return <ChannelsRow title={title} />;
    case "tag":
      return row.tag ? <TagRow tag={row.tag} title={row.title} /> : null;
    default:
      return null;
  }
};

const Promo: React.FC<{ promo: GQL.SiteConfigDataFragment["promo"] }> = ({ promo }) => {
  if (!promo.enabled || !promo.text) return null;
  const inner = (
    <>
      {promo.image && <img src={promo.image} alt="" />}
      <span className="site-promo-text">{promo.text}</span>
      {promo.link && <span className="site-promo-go">{t("Explore ›")}</span>}
    </>
  );
  const external = /^https?:\/\//.test(promo.link);
  return (
    <div className="site-page">
      {promo.link ? (
        external ? (
          <a className="site-promo" href={promo.link} target="_blank" rel="noopener noreferrer">
            {inner}
          </a>
        ) : (
          <Link className="site-promo" to={promo.link}>
            {inner}
          </Link>
        )
      ) : (
        <div className="site-promo">{inner}</div>
      )}
    </div>
  );
};

// ---------------------------------------------------------------- home

export const SiteHome: React.FC = () => {
  const { config, style } = useSite();
  const featured = GQL.useSiteSceneListQuery({
    variables: { kind: GQL.SiteSceneKind.Featured, per_page: 7 },
    fetchPolicy: "cache-and-network",
  });
  const heroScenes = useMemo(
    () => (featured.data?.siteScenes.scenes ?? []).filter((s) => s.paths.screenshot).slice(0, 6),
    [featured.data]
  );

  if (featured.loading && !featured.data) return <Loading />;

  const rows = config.home_rows.filter((r) => r.enabled);

  return (
    <div className="site-home">
      <Hero scenes={heroScenes} style={style} />
      <Ticker />
      <Promo promo={config.promo} />
      <div className="site-page">
        {rows.map((r, i) => (
          <HomeRow key={`${r.type}-${r.tag?.id ?? ""}-${i}`} row={r} />
        ))}
      </div>
    </div>
  );
};

export default SiteHome;
