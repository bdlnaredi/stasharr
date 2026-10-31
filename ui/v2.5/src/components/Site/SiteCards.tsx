import React, { useRef, useState } from "react";
import { Link } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";

export type SiteScene = GQL.SiteSceneCardFragment;
export type SiteModel = GQL.SiteModelCardFragment;

export function sceneTitle(s: Pick<SiteScene, "title" | "files">): string {
  if (s.title) return s.title;
  const path = s.files[0]?.path ?? "";
  const base = path.split(/[\\/]/).pop() ?? "";
  return tidyFileName(base.replace(/\.[^.]+$/, "")) || "Untitled";
}

// Turns a release file name such as
// "Studio.26.10.31.Jane.Doe.XXX.1080p.MP4-GROUP" into "Studio 26 10 31 Jane Doe".
export function tidyFileName(name: string): string {
  let s = name.replace(/[._]+/g, " ").replace(/\s+/g, " ").trim();
  // drop everything from the usual release tags onwards
  const cut = s.search(/\b(XXX|\d{3,4}p|4K|2160p|WEB-?DL|WEBRIP|HEVC|x26[45]|h26[45]|MP4|MKV)\b/i);
  if (cut > 3) s = s.slice(0, cut).trim();
  return s.replace(/[-\s]+$/, "");
}

// A size class for a banner title, so long titles stay readable.
export function titleSize(title: string): string {
  if (title.length > 90) return "site-title-xlong";
  if (title.length > 45) return "site-title-long";
  return "";
}

export function formatTime(seconds?: number | null): string {
  if (!seconds || seconds <= 0) return "";
  const s = Math.round(seconds);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const mm = h > 0 ? String(m).padStart(2, "0") : String(m);
  return `${h > 0 ? `${h}:` : ""}${mm}:${String(sec).padStart(2, "0")}`;
}

export function formatDate(d?: string | null): string {
  if (!d) return "";
  const date = new Date(d.length === 10 ? `${d}T12:00:00` : d);
  if (Number.isNaN(date.getTime())) return d;
  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function qualityBadge(s: Pick<SiteScene, "files">): string | null {
  const f = s.files[0];
  if (!f) return null;
  return resolutionBadge(f.width, f.height);
}

export function resolutionBadge(width: number, height: number): string | null {
  const h = Math.min(width, height) || height;
  if (h >= 2000) return "4K";
  if (h >= 1400) return "2K";
  if (h >= 1000) return "FHD";
  if (h >= 700) return "HD";
  return null;
}

export function isNew(s: SiteScene): boolean {
  const t = new Date(s.created_at).getTime();
  return Date.now() - t < 7 * 24 * 3600 * 1000;
}

export function ageFrom(birthdate?: string | null): number | null {
  if (!birthdate) return null;
  const b = new Date(birthdate);
  if (Number.isNaN(b.getTime())) return null;
  const now = new Date();
  let age = now.getFullYear() - b.getFullYear();
  const m = now.getMonth() - b.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < b.getDate())) age--;
  return age > 0 && age < 120 ? age : null;
}

// A scene thumbnail that plays its short preview when hovered.
export const SceneTile: React.FC<{
  scene: SiteScene;
  size?: "normal" | "large";
}> = ({ scene, size = "normal" }) => {
  const video = useRef<HTMLVideoElement>(null);
  const [playing, setPlaying] = useState(false);
  const timer = useRef<number>();

  const progress =
    scene.resume_time && scene.files[0]?.duration
      ? Math.min(1, scene.resume_time / scene.files[0].duration)
      : 0;

  function start() {
    if (!scene.paths.preview) return;
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      setPlaying(true);
      video.current?.play().catch(() => setPlaying(false));
    }, 250);
  }

  function stop() {
    window.clearTimeout(timer.current);
    setPlaying(false);
    if (video.current) {
      video.current.pause();
      video.current.currentTime = 0;
    }
  }

  const models = scene.performers.filter((p) => p.gender !== GQL.GenderEnum.Male);
  const shown = (models.length ? models : scene.performers).slice(0, 3);
  const badge = qualityBadge(scene);

  return (
    <div className={`site-scene-tile site-scene-tile-${size}`}>
      <Link
        to={`/site/scene/${scene.id}`}
        className="site-thumb"
        onMouseEnter={start}
        onMouseLeave={stop}
        onFocus={start}
        onBlur={stop}
      >
        {scene.paths.screenshot && (
          <img src={scene.paths.screenshot} alt="" loading="lazy" />
        )}
        {scene.paths.preview && (
          <video
            ref={video}
            src={playing ? scene.paths.preview : undefined}
            muted
            loop
            playsInline
            className={playing ? "visible" : ""}
          />
        )}
        <div className="site-thumb-badges">
          {isNew(scene) && <span className="site-badge site-badge-new">New</span>}
          {badge && <span className="site-badge">{badge}</span>}
        </div>
        {scene.files[0]?.duration ? (
          <span className="site-duration">{formatTime(scene.files[0].duration)}</span>
        ) : null}
        {progress > 0.02 && (
          <span className="site-progress">
            <span style={{ width: `${progress * 100}%` }} />
          </span>
        )}
      </Link>
      <div className="site-scene-info">
        <Link to={`/site/scene/${scene.id}`} className="site-scene-title">
          {sceneTitle(scene)}
        </Link>
        {shown.length > 0 && (
          <div className="site-scene-models">
            {shown.map((p, i) => (
              <React.Fragment key={p.id}>
                {i > 0 && ", "}
                <Link to={`/site/model/${p.id}`}>{p.name}</Link>
              </React.Fragment>
            ))}
          </div>
        )}
        <div className="site-scene-meta">
          {scene.studio && (
            <Link to={`/site/studio/${scene.studio.id}`}>{scene.studio.name}</Link>
          )}
          {scene.date && <span>{formatDate(scene.date)}</span>}
        </div>
      </div>
    </div>
  );
};

export const ModelTile: React.FC<{ model: SiteModel; rank?: number }> = ({
  model,
  rank,
}) => (
  <Link to={`/site/model/${model.id}`} className="site-model-tile">
    <div className="site-model-photo">
      {model.image_path && <img src={model.image_path} alt="" loading="lazy" />}
      {rank !== undefined && <span className="site-model-rank">{rank}</span>}
    </div>
    <div className="site-model-name">{model.name}</div>
    <div className="site-model-meta">
      {model.scene_count} scene{model.scene_count === 1 ? "" : "s"}
    </div>
  </Link>
);

export const StudioTile: React.FC<{
  studio: { id: string; name: string; image_path?: string | null; scene_count: number };
}> = ({ studio }) => {
  const hasLogo = studio.image_path && !studio.image_path.includes("default=true");
  return (
    <Link to={`/site/studio/${studio.id}`} className="site-studio-tile">
      <div className="site-studio-logo">
        {hasLogo ? (
          <img src={studio.image_path!} alt={studio.name} loading="lazy" />
        ) : (
          <span>{studio.name}</span>
        )}
      </div>
      <div className="site-studio-name">{studio.name}</div>
      <div className="site-model-meta">{studio.scene_count} scenes</div>
    </Link>
  );
};

export const CategoryTile: React.FC<{
  tag: { id: string; name: string; image_path?: string | null; scene_count: number };
  cover?: string | null;
}> = ({ tag, cover }) => {
  const img =
    tag.image_path && !tag.image_path.includes("default=true") ? tag.image_path : cover;
  return (
    <Link to={`/site/category/${tag.id}`} className="site-category-tile">
      {img && <img src={img} alt="" loading="lazy" />}
      <div className="site-category-label">
        <span>{tag.name}</span>
        <small>{tag.scene_count} scenes</small>
      </div>
    </Link>
  );
};

// A titled block on a page.
export const Section: React.FC<{
  title: React.ReactNode;
  link?: string;
  linkText?: string;
  className?: string;
}> = ({ title, link, linkText = "View all", className, children }) => (
  <section className={`site-section ${className ?? ""}`}>
    <div className="site-section-head">
      <h2>{title}</h2>
      {link && (
        <Link to={link} className="site-section-link">
          {linkText} ›
        </Link>
      )}
    </div>
    {children}
  </section>
);

export const SceneGrid: React.FC<{ scenes: SiteScene[]; large?: boolean }> = ({
  scenes,
  large,
}) => (
  <div className={`site-scene-grid ${large ? "site-scene-grid-large" : ""}`}>
    {scenes.map((s) => (
      <SceneTile key={s.id} scene={s} size={large ? "large" : "normal"} />
    ))}
  </div>
);

// A row that scrolls sideways, with arrow buttons on wide screens.
export const Rail: React.FC<{ className?: string }> = ({ className, children }) => {
  const ref = useRef<HTMLDivElement>(null);
  function scroll(dir: number) {
    const el = ref.current;
    if (el) el.scrollBy({ left: dir * el.clientWidth * 0.85, behavior: "smooth" });
  }
  return (
    <div className={`site-rail ${className ?? ""}`}>
      <button type="button" className="site-rail-arrow left" onClick={() => scroll(-1)} aria-label="Back">
        ‹
      </button>
      <div className="site-rail-track" ref={ref}>
        {children}
      </div>
      <button type="button" className="site-rail-arrow right" onClick={() => scroll(1)} aria-label="More">
        ›
      </button>
    </div>
  );
};

export const Pager: React.FC<{
  page: number;
  total: number;
  perPage: number;
  onPage: (p: number) => void;
}> = ({ page, total, perPage, onPage }) => {
  const pages = Math.max(1, Math.ceil(total / perPage));
  if (pages <= 1) return null;
  const list: (number | "…")[] = [];
  for (let p = 1; p <= pages; p++) {
    if (p === 1 || p === pages || Math.abs(p - page) <= 2) list.push(p);
    else if (list[list.length - 1] !== "…") list.push("…");
  }
  return (
    <nav className="site-pager">
      <button type="button" disabled={page <= 1} onClick={() => onPage(page - 1)}>
        ‹ Prev
      </button>
      {list.map((p, i) =>
        p === "…" ? (
          <span key={`gap-${i}`} className="site-pager-gap">
            …
          </span>
        ) : (
          <button
            type="button"
            key={p}
            className={p === page ? "active" : ""}
            onClick={() => onPage(p)}
          >
            {p}
          </button>
        )
      )}
      <button type="button" disabled={page >= pages} onClick={() => onPage(page + 1)}>
        Next ›
      </button>
    </nav>
  );
};

export const Loading: React.FC = () => (
  <div className="site-loading">
    <span />
    <span />
    <span />
  </div>
);

export const Empty: React.FC<{ text: string }> = ({ text }) => (
  <div className="site-empty">{text}</div>
);
