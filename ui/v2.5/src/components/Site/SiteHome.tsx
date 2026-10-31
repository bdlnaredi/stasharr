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
} from "./SiteCards";
import { ISiteStyle } from "./siteStyles";
import { useSite } from "./SiteContext";

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
        <span>Featured</span>
        {scene.studio && <Link to={`/site/studio/${scene.studio.id}`}>{scene.studio.name}</Link>}
      </div>
      <h1>{sceneTitle(scene)}</h1>
      {(models.length > 0 || scene.performers.length > 0) && (
        <div className="site-hero-models">
          Starring{" "}
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
          ▶ Watch now
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
          {inList ? "✓ In my list" : "+ My list"}
        </button>
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
    const t = window.setTimeout(() => setIndex((i) => (i + 1) % count), 8000);
    return () => window.clearTimeout(t);
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
                aria-label={`Show ${sceneTitle(s)}`}
              />
            ))}
          </div>
        )}
      </div>
      {style.hero === "split" && next.length > 0 && (
        <div className="site-hero-side">
          <div className="site-hero-side-title">Up next</div>
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
      <span className="site-ticker-label">Just added</span>
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

// ---------------------------------------------------------------- home

export const SiteHome: React.FC = () => {
  const { config, style } = useSite();
  const female = config.models_gender === "FEMALE";

  const featured = GQL.useSiteSceneListQuery({
    variables: { kind: GQL.SiteSceneKind.Featured, per_page: 7 },
    fetchPolicy: "cache-and-network",
  });
  const latest = GQL.useSiteScenesQuery({
    variables: { filter: { sort: "created_at", direction: GQL.SortDirectionEnum.Desc, per_page: 12 } },
    fetchPolicy: "cache-and-network",
  });
  const trending = GQL.useSiteSceneListQuery({
    variables: { kind: GQL.SiteSceneKind.Trending, per_page: 12 },
  });
  const topRated = GQL.useSiteSceneListQuery({
    variables: { kind: GQL.SiteSceneKind.TopRated, per_page: 8 },
  });
  const models = GQL.useSiteModelsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 16 },
      performer_filter: female
        ? { gender: { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals } }
        : undefined,
    },
  });
  const studios = GQL.useSiteStudiosQuery({
    variables: { filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 12 } },
  });
  const tags = GQL.useSiteTagsQuery({
    variables: {
      filter: { sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc, per_page: 12 },
      tag_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });
  const extras = GQL.useSiteHomeExtrasQuery({ fetchPolicy: "cache-and-network" });

  const heroScenes = useMemo(
    () => (featured.data?.siteScenes.scenes ?? []).filter((s) => s.paths.screenshot).slice(0, 6),
    [featured.data]
  );
  const latestScenes = latest.data?.findScenes.scenes ?? [];
  const trendingScenes = trending.data?.siteScenes.scenes ?? [];
  const topScenes = topRated.data?.siteScenes.scenes ?? [];
  const modelList = models.data?.findPerformers.performers ?? [];
  const studioList = (studios.data?.findStudios.studios ?? []).filter((s) => s.scene_count > 0);
  const tagList = tags.data?.findTags.tags ?? [];
  const cont = extras.data?.continueWatching ?? [];
  const rec = extras.data?.recommendedScenes ?? [];

  if (featured.loading && !featured.data && latest.loading && !latest.data) {
    return <Loading />;
  }

  return (
    <div className="site-home">
      <Hero scenes={heroScenes} style={style} />
      <Ticker />

      <div className="site-page">
        {cont.length > 0 && (
          <Section title="Continue watching" link="/site/my-list">
            <Rail>
              {cont.map((s) => (
                <SceneTile key={s.id} scene={s} />
              ))}
            </Rail>
          </Section>
        )}

        {latestScenes.length > 0 && (
          <Section title="Latest updates" link="/site/scenes">
            <SceneGrid scenes={latestScenes.slice(0, 8)} />
          </Section>
        )}

        {modelList.length > 0 && (
          <Section title={female ? "Top pornstars" : "Top models"} link="/site/models" className="site-section-models">
            <Rail>
              {modelList.map((m, i) => (
                <ModelTile key={m.id} model={m} rank={i + 1} />
              ))}
            </Rail>
          </Section>
        )}

        {trendingScenes.length > 0 && (
          <Section title="Trending now" link="/site/scenes?sort=trending">
            <Rail>
              {trendingScenes.map((s) => (
                <SceneTile key={s.id} scene={s} />
              ))}
            </Rail>
          </Section>
        )}

        {rec.length > 0 && (
          <Section title="Picked for you">
            <Rail>
              {rec.map((s) => (
                <SceneTile key={s.id} scene={s} />
              ))}
            </Rail>
          </Section>
        )}

        {topScenes.length > 0 && (
          <Section title="Top rated" link="/site/scenes?sort=top">
            <SceneGrid scenes={topScenes.slice(0, 4)} large />
          </Section>
        )}

        {tagList.length > 0 && (
          <Section title="Categories" link="/site/categories">
            <div className="site-category-grid">
              {tagList.map((t) => (
                <CategoryTile key={t.id} tag={t} />
              ))}
            </div>
          </Section>
        )}

        {studioList.length > 0 && (
          <Section title="Channels" link="/site/channels">
            <Rail>
              {studioList.map((s) => (
                <StudioTile key={s.id} studio={s} />
              ))}
            </Rail>
          </Section>
        )}
      </div>
    </div>
  );
};

export default SiteHome;
