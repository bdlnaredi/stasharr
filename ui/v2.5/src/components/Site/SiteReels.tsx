import React, { useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { useCurrentUser } from "src/components/Stasharr/CurrentUser";
import { Empty, Loading, Rail, Section } from "./SiteCards";
import { LikeButton } from "./SiteWidgets";
import { short, t, views } from "./i18n";
import { SiteComments } from "./SiteComments";

type Reel = GQL.ReelDataFragment;

function reelTitle(r: Reel): string {
  if (r.scene.title) return r.scene.title;
  const base = (r.scene.files[0]?.path ?? "").split(/[\\/]/).pop() ?? "";
  return base.replace(/\.[^.]+$/, "").replace(/[._]+/g, " ");
}

// A reel thumbnail for rows: plays muted while hovered.
const ReelCard: React.FC<{ reel: Reel }> = ({ reel }) => {
  const [hover, setHover] = useState(false);
  return (
    <Link
      to={`/site/reels?start=${reel.id}`}
      className="site-reel-card"
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <img src={reel.cover} alt="" loading="lazy" />
      {hover && <video src={reel.stream} autoPlay muted loop playsInline />}
      <span className="site-reel-badge">{t("▶ Reel")}</span>
      {reel.views > 0 && <span className="site-reel-views">👁 {short(reel.views)}</span>}
      <span className="site-reel-card-title">{reelTitle(reel)}</span>
    </Link>
  );
};

export const ReelsRow: React.FC<{ title?: string }> = ({ title }) => {
  const { canViewReels } = useCurrentUser();
  const { data } = GQL.useFindReelsQuery({
    variables: { page: 1, per_page: 16 },
    skip: !canViewReels,
  });
  const reels = data?.findReels.reels ?? [];
  if (!canViewReels || reels.length === 0) return null;
  return (
    <Section title={title || t("Reels")} link="/site/reels" linkText={t("Watch reels")}>
      <Rail className="site-rail-reels">
        {reels.map((r) => (
          <ReelCard key={r.id} reel={r} />
        ))}
      </Rail>
    </Section>
  );
};

// Reels watched recently in this browser tab, so scrolling back and forth
// does not count the same reel again and again.
const counted = new Map<string, number>();
const RECOUNT_AFTER = 30 * 60 * 1000;

// Counts a watch once the reel has played for two seconds.
export function useReelView(id: string, active: boolean, playing: boolean): number | null {
  const [addView] = GQL.useReelAddViewMutation();
  const [total, setTotal] = useState<number | null>(null);
  useEffect(() => {
    if (!active || !playing) return;
    const last = counted.get(id);
    if (last && Date.now() - last < RECOUNT_AFTER) return;
    const timer = window.setTimeout(() => {
      counted.set(id, Date.now());
      addView({ variables: { id } })
        .then((r) => {
          if (r.data) setTotal(r.data.reelAddView);
        })
        .catch(() => counted.delete(id));
    }, 2000);
    return () => window.clearTimeout(timer);
  }, [id, active, playing, addView]);
  return total;
}

const ReelSlide: React.FC<{
  reel: Reel;
  active: boolean;
  muted: boolean;
  onMute: () => void;
  setRef: (el: HTMLDivElement | null) => void;
}> = ({ reel, active, muted, onMute, setRef }) => {
  const video = useRef<HTMLVideoElement>(null);
  const [paused, setPaused] = useState(false);
  const [talk, setTalk] = useState(false);
  const seen = useReelView(reel.id, active, !paused);
  const viewCount = seen ?? reel.views;

  useEffect(() => {
    if (!active) setTalk(false);
  }, [active]);

  useEffect(() => {
    const v = video.current;
    if (!v) return;
    if (active) {
      v.currentTime = 0;
      v.play().then(() => setPaused(false)).catch(() => setPaused(true));
    } else {
      v.pause();
    }
  }, [active]);

  function toggle() {
    const v = video.current;
    if (!v) return;
    if (v.paused) {
      v.play().catch(() => {});
      setPaused(false);
    } else {
      v.pause();
      setPaused(true);
    }
  }

  const models = reel.scene.performers;
  const start = Math.floor(reel.segments[0]?.start ?? 0);

  return (
    <div className="site-reel" ref={setRef} data-reel={reel.id}>
      <video
        ref={video}
        src={active ? reel.stream : undefined}
        poster={reel.cover}
        loop
        playsInline
        muted={muted}
        preload="none"
        onClick={toggle}
      />
      {paused && active && (
        <button type="button" className="site-reel-play" onClick={toggle} aria-label={t("Play")}>
          ▶
        </button>
      )}
      <div className="site-reel-info">
        {reel.scene.studio && (
          <Link to={`/site/studio/${reel.scene.studio.id}`} className="site-kicker">
            {reel.scene.studio.name}
          </Link>
        )}
        <Link to={`/site/scene/${reel.scene.id}`} className="site-reel-title">
          {reelTitle(reel)}
        </Link>
        {models.length > 0 && (
          <div className="site-reel-models">
            {models.slice(0, 3).map((p, i) => (
              <React.Fragment key={p.id}>
                {i > 0 && ", "}
                <Link to={`/site/model/${p.id}`}>{p.name}</Link>
              </React.Fragment>
            ))}
          </div>
        )}
        <div className="site-reel-views-line">👁 {views(viewCount)}</div>
        <Link to={`/site/scene/${reel.scene.id}?t=${start}`} className="site-btn site-btn-primary site-reel-watch">
          {t("▶ Watch full scene")}
        </Link>
      </div>
      <div className="site-reel-side">
        <LikeButton sceneID={reel.scene.id} label={false} className="site-reel-round" />
        <button
          type="button"
          className={`site-btn site-btn-ghost site-reel-round site-reel-talk ${talk ? "on" : ""}`}
          onClick={() => setTalk(!talk)}
          aria-label={t("Comments")}
        >
          💬
          {reel.comment_count > 0 && <small>{short(reel.comment_count)}</small>}
        </button>
        <button type="button" className="site-btn site-btn-ghost site-reel-round" onClick={onMute} aria-label={muted ? t("Unmute") : t("Mute")}>
          {muted ? "🔇" : "🔊"}
        </button>
      </div>
      {talk && (
        <div className="site-reel-comments">
          <div className="site-reel-comments-head">
            <b>{t("Comments")}</b>
            <button type="button" onClick={() => setTalk(false)} aria-label={t("Close")}>
              ✕
            </button>
          </div>
          <SiteComments sceneID={reel.scene.id} reelID={reel.id} compact />
        </div>
      )}
    </div>
  );
};

const perPage = 10;

export const ReelsPage: React.FC = () => {
  const { canViewReels } = useCurrentUser();
  const location = useLocation();
  const startID = useMemo(() => new URLSearchParams(location.search).get("start"), [location.search]);
  const [page, setPage] = useState(1);
  const [reels, setReels] = useState<Reel[]>([]);
  const [active, setActive] = useState<string>();
  const [muted, setMuted] = useState(true);
  const refs = useRef(new Map<string, HTMLDivElement>());
  const observer = useRef<IntersectionObserver>();

  // a new shuffle every visit; the same one while scrolling through it
  const [seed] = useState(() => 1 + Math.floor(Math.random() * 1_000_000));
  const { data, loading } = GQL.useFindReelsQuery({
    variables: { page, per_page: perPage, seed, start_id: startID },
    skip: !canViewReels,
  });
  const total = data?.findReels.count ?? 0;

  useEffect(() => {
    const got = data?.findReels.reels ?? [];
    if (!got.length) return;
    setReels((list) => {
      const seen = new Set(list.map((r) => r.id));
      let next = [...list, ...got.filter((r) => !seen.has(r.id))];
      // a reel picked from a row goes first
      if (page === 1 && startID) {
        const i = next.findIndex((r) => r.id === startID);
        if (i > 0) next = [next[i], ...next.slice(0, i), ...next.slice(i + 1)];
      }
      return next;
    });
  }, [data, page, startID]);

  useEffect(() => {
    if (!active && reels.length) setActive(reels[0].id);
  }, [reels, active]);

  useEffect(() => {
    observer.current = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting && e.intersectionRatio > 0.6) {
            setActive((e.target as HTMLElement).dataset.reel);
          }
        }
      },
      { threshold: [0.6] }
    );
    refs.current.forEach((el) => observer.current?.observe(el));
    return () => observer.current?.disconnect();
  }, []);

  // load more near the end
  useEffect(() => {
    const i = reels.findIndex((r) => r.id === active);
    if (i >= reels.length - 3 && reels.length < total && !loading) {
      setPage((p) => p + 1);
    }
  }, [active, reels, total, loading]);

  // arrow keys move between reels
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
      const i = reels.findIndex((r) => r.id === active);
      const next = reels[i + (e.key === "ArrowDown" ? 1 : -1)];
      if (next) {
        e.preventDefault();
        refs.current.get(next.id)?.scrollIntoView({ behavior: "smooth" });
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [reels, active]);

  if (!canViewReels) {
    return (
      <div className="site-page">
        <Empty text={t("Reels are not available for your account.")} />
      </div>
    );
  }
  if (loading && reels.length === 0) return <Loading />;
  if (reels.length === 0) {
    return (
      <div className="site-page">
        <Empty text={t("No reels yet.")} />
      </div>
    );
  }

  return (
    <div className="site-reels">
      {reels.map((r) => (
        <ReelSlide
          key={r.id}
          reel={r}
          active={r.id === active}
          muted={muted}
          onMute={() => setMuted(!muted)}
          setRef={(el) => {
            const old = refs.current.get(r.id);
            if (old && old !== el) observer.current?.unobserve(old);
            if (el) {
              refs.current.set(r.id, el);
              observer.current?.observe(el);
            } else {
              refs.current.delete(r.id);
            }
          }}
        />
      ))}
    </div>
  );
};
