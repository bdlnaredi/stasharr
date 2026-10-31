import React, { useEffect, useMemo, useRef, useState } from "react";
import { Link, useHistory, useLocation, useParams } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { ScenePlayer } from "src/components/ScenePlayer/ScenePlayer";
import {
  CategoryTile,
  Empty,
  Loading,
  ModelTile,
  Pager,
  Rail,
  SceneGrid,
  SceneTile,
  Section,
  StudioTile,
  ageFrom,
  formatDate,
  formatTime,
  resolutionBadge,
  sceneTitle,
  titleSize,
} from "./SiteCards";
import { useSite } from "./SiteContext";
import { FollowButton, LikeButton, SceneRating } from "./SiteWidgets";

function useQueryParams(): [URLSearchParams, (changes: Record<string, string | null>) => void] {
  const location = useLocation();
  const history = useHistory();
  const params = useMemo(() => new URLSearchParams(location.search), [location.search]);
  function set(changes: Record<string, string | null>) {
    const next = new URLSearchParams(location.search);
    for (const [k, v] of Object.entries(changes)) {
      if (v === null || v === "") next.delete(k);
      else next.set(k, v);
    }
    history.push({ pathname: location.pathname, search: next.toString() });
    window.scrollTo({ top: 0 });
  }
  return [params, set];
}

function pageParam(params: URLSearchParams): number {
  const p = Number.parseInt(params.get("page") ?? "1", 10);
  return Number.isNaN(p) || p < 1 ? 1 : p;
}

// ---------------------------------------------------------------- scene lists

interface ISortOption {
  id: string;
  label: string;
  // either a normal scene sort or a list counted across all users
  sort?: string;
  direction?: GQL.SortDirectionEnum;
  kind?: GQL.SiteSceneKind;
}

const allSorts: ISortOption[] = [
  { id: "latest", label: "Latest", sort: "created_at", direction: GQL.SortDirectionEnum.Desc },
  { id: "trending", label: "Trending", kind: GQL.SiteSceneKind.Trending },
  { id: "popular", label: "Most viewed", kind: GQL.SiteSceneKind.Popular },
  { id: "top", label: "Top rated", kind: GQL.SiteSceneKind.TopRated },
  { id: "release", label: "Release date", sort: "date", direction: GQL.SortDirectionEnum.Desc },
  { id: "longest", label: "Longest", sort: "duration", direction: GQL.SortDirectionEnum.Desc },
  { id: "random", label: "Random", sort: "random", direction: GQL.SortDirectionEnum.Asc },
];

const PER_PAGE = 24;

// A sortable, paged grid of scenes. Lists counted across all users are only
// offered when there is no filter.
const lengths: { id: string; label: string; f: GQL.IntCriterionInput }[] = [
  { id: "short", label: "Under 10 min", f: { value: 600, modifier: GQL.CriterionModifier.LessThan } },
  { id: "medium", label: "10–30 min", f: { value: 600, value2: 1800, modifier: GQL.CriterionModifier.Between } },
  { id: "long", label: "30–60 min", f: { value: 1800, value2: 3600, modifier: GQL.CriterionModifier.Between } },
  { id: "xlong", label: "Over 1 hour", f: { value: 3600, modifier: GQL.CriterionModifier.GreaterThan } },
];

const qualities: { id: string; label: string; f: GQL.ResolutionCriterionInput }[] = [
  { id: "4k", label: "4K", f: { value: GQL.ResolutionEnum.QuadHd, modifier: GQL.CriterionModifier.GreaterThan } },
  { id: "fhd", label: "1080p and up", f: { value: GQL.ResolutionEnum.StandardHd, modifier: GQL.CriterionModifier.GreaterThan } },
  { id: "hd", label: "720p and up", f: { value: GQL.ResolutionEnum.Standard, modifier: GQL.CriterionModifier.GreaterThan } },
];

// Category, length, quality and channel pickers above a scene list.
const SceneFilters: React.FC<{ hideCategory?: boolean; hideChannel?: boolean }> = ({
  hideCategory,
  hideChannel,
}) => {
  const [params, set] = useQueryParams();
  const { data } = GQL.useSiteFilterOptionsQuery({ fetchPolicy: "cache-first" });
  const tags = data?.findTags.tags ?? [];
  const studios = data?.findStudios.studios ?? [];
  const active = ["cat", "len", "res", "ch"].some((k) => params.get(k));

  function select(key: string, label: string, options: { id: string; label: string }[]) {
    return (
      <label className="site-filter">
        <span>{label}</span>
        <select value={params.get(key) ?? ""} onChange={(e) => set({ [key]: e.currentTarget.value || null, page: null })}>
          <option value="">Any</option>
          {options.map((o) => (
            <option key={o.id} value={o.id}>
              {o.label}
            </option>
          ))}
        </select>
      </label>
    );
  }

  return (
    <div className="site-filters">
      {!hideCategory && select("cat", "Category", tags.map((t) => ({ id: t.id, label: t.name })))}
      {select("len", "Length", lengths)}
      {select("res", "Quality", qualities)}
      {!hideChannel && select("ch", "Channel", studios.map((t) => ({ id: t.id, label: t.name })))}
      {active && (
        <button type="button" className="site-filter-clear" onClick={() => set({ cat: null, len: null, res: null, ch: null, page: null })}>
          Clear filters
        </button>
      )}
    </div>
  );
};

function filtersFromParams(params: URLSearchParams, base?: GQL.SceneFilterType): GQL.SceneFilterType | undefined {
  const f: GQL.SceneFilterType = { ...(base ?? {}) };
  let any = !!base;
  const cat = params.get("cat");
  if (cat) {
    f.tags = { value: [cat], modifier: GQL.CriterionModifier.Includes, depth: 0 };
    any = true;
  }
  const len = lengths.find((l) => l.id === params.get("len"));
  if (len) {
    f.duration = len.f;
    any = true;
  }
  const res = qualities.find((r) => r.id === params.get("res"));
  if (res) {
    f.resolution = res.f;
    any = true;
  }
  const ch = params.get("ch");
  if (ch) {
    f.studios = { value: [ch], modifier: GQL.CriterionModifier.Includes, depth: -1 };
    any = true;
  }
  return any ? f : undefined;
}

// A sortable, filterable, paged grid of scenes. Lists counted across all
// users are only offered when nothing is filtered.
const SceneBrowser: React.FC<{
  sceneFilter?: GQL.SceneFilterType;
  q?: string;
  hideCategory?: boolean;
  hideChannel?: boolean;
}> = ({ sceneFilter, q, hideCategory, hideChannel }) => {
  const [params, set] = useQueryParams();
  const filter = filtersFromParams(params, sceneFilter);
  const sorts = filter || q ? allSorts.filter((s) => !s.kind) : allSorts;
  const current = sorts.find((s) => s.id === params.get("sort")) ?? sorts[0];
  const page = pageParam(params);
  const [seed] = useState(() => Math.floor(Math.random() * 1e8));

  const list = GQL.useSiteSceneListQuery({
    variables: { kind: current.kind ?? GQL.SiteSceneKind.Popular, page, per_page: PER_PAGE },
    skip: !current.kind,
  });
  const find = GQL.useSiteScenesQuery({
    variables: {
      filter: {
        q,
        page,
        per_page: PER_PAGE,
        sort: current.sort === "random" ? `random_${seed}` : current.sort,
        direction: current.direction,
      },
      scene_filter: filter,
    },
    skip: !!current.kind,
  });

  const loading = current.kind ? list.loading : find.loading;
  const scenes = (current.kind ? list.data?.siteScenes.scenes : find.data?.findScenes.scenes) ?? [];
  const total = (current.kind ? list.data?.siteScenes.count : find.data?.findScenes.count) ?? 0;

  return (
    <>
      <div className="site-toolbar">
        <div className="site-tabs">
          {sorts.map((s) => (
            <button
              type="button"
              key={s.id}
              className={s.id === current.id ? "active" : ""}
              onClick={() => set({ sort: s.id === sorts[0].id ? null : s.id, page: null })}
            >
              {s.label}
            </button>
          ))}
        </div>
        <span className="site-count">{total.toLocaleString()} scenes</span>
      </div>
      <SceneFilters hideCategory={hideCategory} hideChannel={hideChannel} />
      {loading && scenes.length === 0 ? (
        <Loading />
      ) : scenes.length === 0 ? (
        <Empty text="No scenes match." />
      ) : (
        <SceneGrid scenes={scenes} />
      )}
      <Pager page={page} total={total} perPage={PER_PAGE} onPage={(p) => set({ page: String(p) })} />
    </>
  );
};

export const ScenesPage: React.FC = () => (
  <div className="site-page">
    <h1 className="site-page-title">All scenes</h1>
    <SceneBrowser />
  </div>
);

export const SearchPage: React.FC = () => {
  const [params] = useQueryParams();
  const q = params.get("q") ?? "";
  const { config } = useSite();
  const models = GQL.useSiteModelsQuery({
    variables: {
      filter: { q, per_page: 12, sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc },
      performer_filter:
        config.models_gender === "FEMALE"
          ? { gender: { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals } }
          : undefined,
    },
    skip: !q,
  });
  const found = models.data?.findPerformers.performers ?? [];
  return (
    <div className="site-page">
      <h1 className="site-page-title">
        Results for <em>“{q}”</em>
      </h1>
      {found.length > 0 && (
        <Section title="Models">
          <Rail>
            {found.map((m) => (
              <ModelTile key={m.id} model={m} />
            ))}
          </Rail>
        </Section>
      )}
      <SceneBrowser q={q} />
    </div>
  );
};

// ---------------------------------------------------------------- scene page

// Shown when a scene ends: plays the next one after a countdown.
const UpNext: React.FC<{ next: GQL.SiteSceneCardFragment; onCancel: () => void }> = ({ next, onCancel }) => {
  const history = useHistory();
  const [left, setLeft] = useState(8);
  useEffect(() => {
    if (left <= 0) {
      history.push(`/site/scene/${next.id}`);
      return;
    }
    const t = window.setTimeout(() => setLeft(left - 1), 1000);
    return () => window.clearTimeout(t);
  }, [left, next.id, history]);
  return (
    <div className="site-upnext">
      {next.paths.screenshot && <img src={next.paths.screenshot} alt="" />}
      <div className="site-upnext-text">
        <span className="site-kicker">Up next in {left}s</span>
        <b>{sceneTitle(next)}</b>
        <div>
          <button type="button" className="site-btn site-btn-primary" onClick={() => history.push(`/site/scene/${next.id}`)}>
            ▶ Play now
          </button>{" "}
          <button type="button" className="site-btn site-btn-ghost" onClick={onCancel}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};

function readTheatre(): boolean {
  try {
    return localStorage.getItem("stasharr-site-theatre") === "1";
  } catch {
    return false;
  }
}

function saveTheatre(on: boolean) {
  try {
    localStorage.setItem("stasharr-site-theatre", on ? "1" : "0");
  } catch {
    // private window: not remembered
  }
}

export const ScenePage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const { data, loading } = GQL.useFindSceneQuery({ variables: { id } });
  const similar = GQL.useSiteSimilarQuery({ variables: { scene_id: id } });
  const [toggle] = GQL.useWatchLaterToggleMutation({ refetchQueries: ["SiteMyList"] });
  const list = GQL.useScenePlaylistsQuery({ variables: { id }, fetchPolicy: "cache-and-network" });
  const [inList, setInList] = useState(false);
  const [ended, setEnded] = useState(false);
  const [theatre, setTheatre] = useState(readTheatre);
  const seek = useRef<(value: number) => void>();
  useEffect(() => setInList(!!list.data?.findScene?.in_watch_later), [list.data]);
  useEffect(() => setEnded(false), [id]);
  const scene = data?.findScene;

  const mainModel = scene?.performers.find((p) => p.gender !== GQL.GenderEnum.Male) ?? scene?.performers[0];
  const more = GQL.useSiteScenesQuery({
    variables: {
      filter: { per_page: 9, sort: "random", direction: GQL.SortDirectionEnum.Asc },
      scene_filter: mainModel
        ? { performers: { value: [mainModel.id], modifier: GQL.CriterionModifier.Includes } }
        : undefined,
    },
    skip: !mainModel,
  });

  useEffect(() => window.scrollTo({ top: 0 }), [id]);

  // ?t= starts at a time, for example from a reel
  const startAt = useMemo(() => {
    const t = Number(new URLSearchParams(location.search).get("t"));
    return Number.isFinite(t) && t > 0 ? t : null;
  }, [location.search]);

  const initial = useMemo(() => {
    if (startAt) return startAt;
    const resume = scene?.resume_time ?? 0;
    const duration = scene?.files[0]?.duration ?? 0;
    return resume > 10 && resume < duration - 30 ? resume : 0;
  }, [scene, startAt]);

  if (loading && !scene) return <Loading />;
  if (!scene) {
    return (
      <div className="site-page">
        <Empty text="This scene is not available." />
      </div>
    );
  }

  const models = scene.performers.filter((p) => p.gender !== GQL.GenderEnum.Male);
  const others = scene.performers.filter((p) => p.gender === GQL.GenderEnum.Male);
  const moreScenes = (more.data?.findScenes.scenes ?? []).filter((s) => s.id !== scene.id).slice(0, 8);
  const similarScenes = similar.data?.findSimilarScenes ?? [];
  const next = similarScenes[0] ?? moreScenes[0];
  const f = scene.files[0];
  const badge = f ? resolutionBadge(f.width, f.height) : null;
  const chapters = [...scene.scene_markers].sort((a, b) => a.seconds - b.seconds);
  const duration = f?.duration ?? 0;

  function jump(seconds: number) {
    seek.current?.(seconds);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  return (
    <div className={`site-scene-page ${theatre ? "theatre" : ""}`}>
      <div className="site-player">
        <ScenePlayer
          key={scene.id}
          scene={scene}
          hideScrubberOverride
          autoplay
          permitLoop={false}
          initialTimestamp={initial}
          sendSetTimestamp={(fn) => {
            seek.current = fn;
          }}
          onComplete={() => setEnded(true)}
          onNext={() => next && setEnded(true)}
          onPrevious={() => {}}
        />
        {ended && next && <UpNext next={next} onCancel={() => setEnded(false)} />}
      </div>

      {chapters.length > 0 && duration > 0 && (
        <div className="site-chapters">
          <div className="site-chapter-bar">
            {chapters.map((c) => (
              <button
                type="button"
                key={c.id}
                style={{ left: `${Math.min(100, (c.seconds / duration) * 100)}%` }}
                title={`${formatTime(c.seconds)} ${c.title || c.primary_tag.name}`}
                onClick={() => jump(c.seconds)}
              />
            ))}
          </div>
          <div className="site-chapter-list">
            {chapters.map((c) => (
              <button type="button" key={c.id} className="site-chip" onClick={() => jump(c.seconds)}>
                <b>{formatTime(c.seconds) || "0:00"}</b> {c.title || c.primary_tag.name}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="site-page">
        <div className="site-scene-header">
          <div className="site-scene-header-main">
            {scene.studio && (
              <Link to={`/site/studio/${scene.studio.id}`} className="site-kicker">
                {scene.studio.name}
              </Link>
            )}
            <h1 className={titleSize(sceneTitle(scene))}>{sceneTitle(scene)}</h1>
            <div className="site-hero-meta">
              {scene.date && <span>{formatDate(scene.date)}</span>}
              {f?.duration ? <span>{formatTime(f.duration)}</span> : null}
              {badge && <span className="site-badge">{badge}</span>}
            </div>
            <SceneRating sceneID={scene.id} />
          </div>
          <div className="site-scene-actions">
            <LikeButton sceneID={scene.id} />
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
            <button
              type="button"
              className="site-btn site-btn-ghost site-theatre-btn"
              onClick={() => {
                saveTheatre(!theatre);
                setTheatre(!theatre);
              }}
            >
              {theatre ? "▣ Normal view" : "▭ Theatre mode"}
            </button>
          </div>
        </div>

        {scene.performers.length > 0 && (
          <div className="site-cast">
            {[...models, ...others].map((p) => (
              <Link key={p.id} to={`/site/model/${p.id}`} className="site-cast-item">
                <span className="site-cast-photo">
                  {p.image_path && <img src={p.image_path} alt="" loading="lazy" />}
                </span>
                <span>{p.name}</span>
              </Link>
            ))}
          </div>
        )}

        {scene.details && <p className="site-scene-details">{scene.details}</p>}

        {scene.tags.length > 0 && (
          <div className="site-tag-list">
            {scene.tags.map((t) => (
              <Link key={t.id} to={`/site/category/${t.id}`} className="site-chip">
                {t.name}
              </Link>
            ))}
          </div>
        )}

        {moreScenes.length > 0 && mainModel && (
          <Section title={`More of ${mainModel.name}`} link={`/site/model/${mainModel.id}`}>
            <SceneGrid scenes={moreScenes} />
          </Section>
        )}

        {similarScenes.length > 0 && (
          <Section title="You may also like">
            <Rail>
              {similarScenes.map((s) => (
                <SceneTile key={s.id} scene={s} />
              ))}
            </Rail>
          </Section>
        )}
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- models

const modelSorts = [
  { id: "popular", label: "Most scenes", sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc },
  { id: "rating", label: "Top rated", sort: "rating", direction: GQL.SortDirectionEnum.Desc },
  { id: "new", label: "Newest", sort: "created_at", direction: GQL.SortDirectionEnum.Desc },
  { id: "name", label: "A–Z", sort: "name", direction: GQL.SortDirectionEnum.Asc },
];

export const ModelsPage: React.FC = () => {
  const { config } = useSite();
  const [params, set] = useQueryParams();
  const current = modelSorts.find((s) => s.id === params.get("sort")) ?? modelSorts[0];
  const page = pageParam(params);
  const letter = params.get("letter") ?? "";
  const per = 36;

  const filter: GQL.PerformerFilterType = {
    scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan },
  };
  if (config.models_gender === "FEMALE") {
    filter.gender = { value: GQL.GenderEnum.Female, modifier: GQL.CriterionModifier.Equals };
  }
  if (letter) {
    filter.name = { value: `^${letter}`, modifier: GQL.CriterionModifier.MatchesRegex };
  }

  const { data, loading } = GQL.useSiteModelsQuery({
    variables: {
      filter: { page, per_page: per, sort: current.sort, direction: current.direction },
      performer_filter: filter,
    },
  });
  const models = data?.findPerformers.performers ?? [];
  const total = data?.findPerformers.count ?? 0;

  return (
    <div className="site-page">
      <h1 className="site-page-title">{config.models_gender === "FEMALE" ? "Pornstars" : "Models"}</h1>
      <div className="site-toolbar">
        <div className="site-tabs">
          {modelSorts.map((s) => (
            <button
              type="button"
              key={s.id}
              className={s.id === current.id ? "active" : ""}
              onClick={() => set({ sort: s.id === "popular" ? null : s.id, page: null })}
            >
              {s.label}
            </button>
          ))}
        </div>
        <span className="site-count">{total.toLocaleString()} models</span>
      </div>
      <div className="site-letters">
        <button type="button" className={!letter ? "active" : ""} onClick={() => set({ letter: null, page: null })}>
          All
        </button>
        {"ABCDEFGHIJKLMNOPQRSTUVWXYZ".split("").map((l) => (
          <button
            type="button"
            key={l}
            className={letter === l ? "active" : ""}
            onClick={() => set({ letter: l, page: null })}
          >
            {l}
          </button>
        ))}
      </div>
      {loading && models.length === 0 ? (
        <Loading />
      ) : models.length === 0 ? (
        <Empty text="No models found." />
      ) : (
        <div className="site-model-grid">
          {models.map((m) => (
            <ModelTile key={m.id} model={m} />
          ))}
        </div>
      )}
      <Pager page={page} total={total} perPage={per} onPage={(p) => set({ page: String(p) })} />
    </div>
  );
};

function heightText(cm?: number | null): string | null {
  if (!cm) return null;
  const inches = Math.round(cm / 2.54);
  return `${cm} cm (${Math.floor(inches / 12)}′${inches % 12}″)`;
}

export const ModelPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { data, loading } = GQL.useSiteModelQuery({ variables: { id } });
  const popular = GQL.useSiteSceneListQuery({
    variables: { kind: GQL.SiteSceneKind.Popular, per_page: 8, performer_id: id },
  });
  const costars = GQL.useSiteCoStarsQuery({ variables: { performer_id: id } });
  const m = data?.findPerformer;
  const filter = useMemo<GQL.SceneFilterType>(
    () => ({ performers: { value: [id], modifier: GQL.CriterionModifier.Includes } }),
    [id]
  );

  if (loading && !m) return <Loading />;
  if (!m) {
    return (
      <div className="site-page">
        <Empty text="This model is not available." />
      </div>
    );
  }

  const age = ageFrom(m.birthdate);
  const facts: [string, string | null | undefined][] = [
    ["Age", age ? String(age) : null],
    ["From", m.country],
    ["Ethnicity", m.ethnicity],
    ["Hair", m.hair_color],
    ["Eyes", m.eye_color],
    ["Height", heightText(m.height_cm)],
    ["Measurements", m.measurements],
    ["Career", m.career_start ? `${m.career_start}${m.career_end ? ` – ${m.career_end}` : " – now"}` : null],
    ["Tattoos", m.tattoos],
    ["Piercings", m.piercings],
  ];
  const popularScenes = popular.data?.siteScenes.scenes ?? [];
  // a wide picture from her most watched scene makes the cover
  const cover = popularScenes.find((s) => s.paths.screenshot)?.paths.screenshot ?? m.image_path;
  const costarList = costars.data?.siteCoStars ?? [];

  return (
    <div className="site-model-page">
      <div className="site-model-hero">
        {cover && <div className="site-model-cover" style={{ backgroundImage: `url("${cover}")` }} />}
        <div className="site-model-cover-shade" />
        <div className="site-page site-model-hero-inner">
          <div className="site-model-hero-photo">{m.image_path && <img src={m.image_path} alt={m.name} />}</div>
          <div className="site-model-hero-text">
            <span className="site-kicker">{m.gender === GQL.GenderEnum.Female ? "Pornstar" : "Model"}</span>
            <h1>{m.name}</h1>
            {m.alias_list.length > 0 && <div className="site-model-aliases">Also known as {m.alias_list.slice(0, 5).join(", ")}</div>}
            <div className="site-model-stat">
              <strong>{m.scene_count}</strong> scene{m.scene_count === 1 ? "" : "s"}
            </div>
            <FollowButton performerID={m.id} following={!!m.site_favorite} />
            <dl className="site-facts">
              {facts
                .filter(([, v]) => !!v)
                .map(([k, v]) => (
                  <React.Fragment key={k}>
                    <dt>{k}</dt>
                    <dd>{v}</dd>
                  </React.Fragment>
                ))}
            </dl>
            {m.details && <p className="site-model-bio">{m.details}</p>}
          </div>
        </div>
      </div>
      <div className="site-page">
        {popularScenes.length > 1 && (
          <Section title="Most popular scenes">
            <Rail>
              {popularScenes.map((s) => (
                <SceneTile key={s.id} scene={s} />
              ))}
            </Rail>
          </Section>
        )}
        {costarList.length > 0 && (
          <Section title="Frequently seen with" className="site-section-models">
            <Rail>
              {costarList.map((c) => (
                <Link key={c.performer.id} to={`/site/model/${c.performer.id}`} className="site-model-tile">
                  <div className="site-model-photo">
                    {c.performer.image_path && <img src={c.performer.image_path} alt="" loading="lazy" />}
                  </div>
                  <div className="site-model-name">{c.performer.name}</div>
                  <div className="site-model-meta">
                    {c.scenes} scene{c.scenes === 1 ? "" : "s"} together
                  </div>
                </Link>
              ))}
            </Rail>
          </Section>
        )}
        <h2 className="site-page-subtitle">All scenes with {m.name}</h2>
        <SceneBrowser sceneFilter={filter} />
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- channels

export const ChannelsPage: React.FC = () => {
  const [params, set] = useQueryParams();
  const page = pageParam(params);
  const per = 36;
  const { data, loading } = GQL.useSiteStudiosQuery({
    variables: {
      filter: { page, per_page: per, sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc },
      studio_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });
  const studios = data?.findStudios.studios ?? [];
  return (
    <div className="site-page">
      <h1 className="site-page-title">Channels</h1>
      {loading && studios.length === 0 ? (
        <Loading />
      ) : (
        <div className="site-studio-grid">
          {studios.map((s) => (
            <StudioTile key={s.id} studio={s} />
          ))}
        </div>
      )}
      <Pager page={page} total={data?.findStudios.count ?? 0} perPage={per} onPage={(p) => set({ page: String(p) })} />
    </div>
  );
};

export const ChannelPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { data } = GQL.useSiteStudioQuery({ variables: { id } });
  const s = data?.findStudio;
  const filter = useMemo<GQL.SceneFilterType>(
    () => ({ studios: { value: [id], modifier: GQL.CriterionModifier.Includes, depth: -1 } }),
    [id]
  );
  const hasLogo = s?.image_path && !s.image_path.includes("default=true");
  return (
    <div className="site-page">
      <div className="site-channel-head">
        {hasLogo && <img src={s!.image_path!} alt="" />}
        <div>
          <span className="site-kicker">Channel</span>
          <h1>{s?.name ?? ""}</h1>
          {s?.details && <p>{s.details}</p>}
        </div>
      </div>
      <SceneBrowser sceneFilter={filter} hideChannel />
    </div>
  );
};

// ---------------------------------------------------------------- categories

export const CategoriesPage: React.FC = () => {
  const [params, set] = useQueryParams();
  const page = pageParam(params);
  const per = 48;
  const { data, loading } = GQL.useSiteTagsQuery({
    variables: {
      filter: { page, per_page: per, sort: "scenes_count", direction: GQL.SortDirectionEnum.Desc },
      tag_filter: { scene_count: { value: 0, modifier: GQL.CriterionModifier.GreaterThan } },
    },
  });
  const tags = data?.findTags.tags ?? [];
  return (
    <div className="site-page">
      <h1 className="site-page-title">Categories</h1>
      {loading && tags.length === 0 ? (
        <Loading />
      ) : (
        <div className="site-category-grid site-category-grid-all">
          {tags.map((t) => (
            <CategoryTile key={t.id} tag={t} />
          ))}
        </div>
      )}
      <Pager page={page} total={data?.findTags.count ?? 0} perPage={per} onPage={(p) => set({ page: String(p) })} />
    </div>
  );
};

export const CategoryPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { data } = GQL.useSiteTagQuery({ variables: { id } });
  const t = data?.findTag;
  const filter = useMemo<GQL.SceneFilterType>(
    () => ({ tags: { value: [id], modifier: GQL.CriterionModifier.Includes, depth: 0 } }),
    [id]
  );
  return (
    <div className="site-page">
      <span className="site-kicker">Category</span>
      <h1 className="site-page-title">{t?.name ?? ""}</h1>
      {t?.description && <p className="site-lead">{t.description}</p>}
      <SceneBrowser sceneFilter={filter} hideCategory />
    </div>
  );
};

// ---------------------------------------------------------------- my list

export const MyListPage: React.FC = () => {
  const lists = GQL.useSiteMyListQuery({ fetchPolicy: "cache-and-network" });
  const extras = GQL.useSiteHomeExtrasQuery({ fetchPolicy: "cache-and-network" });
  const liked = GQL.useSiteLikedQuery({ variables: { per_page: 48 }, fetchPolicy: "cache-and-network" });
  const favs = GQL.useSiteFavoriteModelsQuery({ fetchPolicy: "cache-and-network" });
  const fromFavs = GQL.useSiteFromFavoritesQuery({ variables: { per_page: 12 }, fetchPolicy: "cache-and-network" });
  const later = lists.data?.findPlaylists.find((p) => p.watch_later)?.scenes ?? [];
  const cont = extras.data?.continueWatching ?? [];
  const likedScenes = liked.data?.siteLikedScenes.scenes ?? [];
  const favModels = favs.data?.siteFavoriteModels ?? [];
  const newFromFavs = fromFavs.data?.siteFromFavorites.scenes ?? [];

  if (lists.loading && !lists.data) return <Loading />;

  return (
    <div className="site-page">
      <h1 className="site-page-title">My list</h1>
      {cont.length > 0 && (
        <Section title="Continue watching">
          <Rail>
            {cont.map((s) => (
              <SceneTile key={s.id} scene={s} />
            ))}
          </Rail>
        </Section>
      )}
      <Section title="Favourite pornstars" className="site-section-models">
        {favModels.length === 0 ? (
          <Empty text="Tap “Add to favourites” on a model's page to follow her." />
        ) : (
          <Rail>
            {favModels.map((m) => (
              <ModelTile key={m.id} model={m} />
            ))}
          </Rail>
        )}
      </Section>
      {newFromFavs.length > 0 && (
        <Section title="New from your favourites">
          <Rail>
            {newFromFavs.map((s) => (
              <SceneTile key={s.id} scene={s} />
            ))}
          </Rail>
        </Section>
      )}
      <Section title="Saved for later">
        {later.length === 0 ? <Empty text="Nothing saved yet. Tap “+ My list” on any scene." /> : <SceneGrid scenes={later} />}
      </Section>
      <Section title="Liked scenes">
        {likedScenes.length === 0 ? <Empty text="Tap the heart on a scene to like it." /> : <SceneGrid scenes={likedScenes} />}
      </Section>
    </div>
  );
};
