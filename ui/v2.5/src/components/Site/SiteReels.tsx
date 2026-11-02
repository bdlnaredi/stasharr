import React, { useEffect, useMemo, useRef, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import * as GQL from "src/core/generated-graphql";
import { useCurrentUser } from "src/components/Stasharr/CurrentUser";
import { Empty, Loading, Rail, Section } from "./SiteCards";
import { LikeButton } from "./SiteWidgets";
import { t } from "./i18n";

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
      {reel.scene.paths.screenshot && <img src={reel.scene.paths.screenshot} alt="" loading="lazy" />}
      {hover && <video src={reel.stream} autoPlay muted loop playsInline />}
      <span className="site-reel-badge">{t("▶ Reel")}</span>
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

const ReelSlide: React.FC<{
  reel: Reel;
  active: boolean;
  muted: boolean;
  onMute: () => void;
  setRef: (el: HTMLDivElement | null) => void;
}> = ({ reel, active, muted, onMute, setRef }) => {
  const video = useRef<HTMLVideoElement>(null);
  const [paused, setPaused] = useState(false);

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
        poster={reel.scene.paths.screenshot ?? undefined}
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
        <Link to={`/site/scene/${reel.scene.id}?t=${start}`} className="site-btn site-btn-primary site-reel-watch">
          {t("▶ Watch full scene")}
        </Link>
      </div>
      <div className="site-reel-side">
        <LikeButton sceneID={reel.scene.id} label={false} className="site-reel-round" />
        <button type="button" className="site-btn site-btn-ghost site-reel-round" onClick={onMute} aria-label={muted ? t("Unmute") : t("Mute")}>
          {muted ? "🔇" : "🔊"}
        </button>
      </div>
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

  const { data, loading } = GQL.useFindReelsQuery({
    variables: { page, per_page: perPage },
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
