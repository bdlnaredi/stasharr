import React, { useCallback, useEffect, useRef, useState } from "react";
import { Button } from "react-bootstrap";
import { Link } from "react-router-dom";
import { Helmet } from "react-helmet";
import {
  faRandom,
  faRedo,
  faTrashAlt,
  faVolumeMute,
  faVolumeUp,
} from "@fortawesome/free-solid-svg-icons";
import * as GQL from "src/core/generated-graphql";
import { Icon } from "src/components/Shared/Icon";
import { LoadingIndicator } from "src/components/Shared/LoadingIndicator";
import { useToast } from "src/hooks/Toast";
import { useCurrentUser } from "./CurrentUser";

type Reel = GQL.ReelDataFragment;

const perPage = 12;

function sceneTitle(reel: Reel) {
  const s = reel.scene;
  if (s.title) return s.title;
  const path = s.files[0]?.path ?? "";
  return path.split(/[\\/]/).pop() ?? "";
}

function sourceLabel(source: string) {
  switch (source) {
    case "MARKERS":
      return "from markers";
    case "HEATMAP":
      return "from interactive heatmap";
    case "ANALYSIS":
      return "from audio and motion";
    default:
      return "evenly spaced";
  }
}

const ReelItem: React.FC<{
  reel: Reel;
  active: boolean;
  muted: boolean;
  canEdit: boolean;
  onToggleMute: () => void;
  onDelete: (reel: Reel) => void;
  onRemake: (reel: Reel) => void;
  setRef: (el: HTMLDivElement | null) => void;
}> = ({ reel, active, muted, canEdit, onToggleMute, onDelete, onRemake, setRef }) => {
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    if (active) {
      v.currentTime = 0;
      v.play().catch(() => {
        // autoplay with sound can be blocked until the user interacts
      });
    } else {
      v.pause();
    }
  }, [active]);

  function togglePlay() {
    const v = videoRef.current;
    if (!v) return;
    if (v.paused) {
      v.play().catch(() => {});
    } else {
      v.pause();
    }
  }

  const performers = reel.scene.performers.map((p) => p.name).join(", ");

  return (
    <div className="stasharr-reel" ref={setRef} data-reel-id={reel.id}>
      <video
        ref={videoRef}
        className="stasharr-reel-video"
        src={active ? reel.stream : undefined}
        poster={reel.scene.paths.screenshot ?? undefined}
        loop
        playsInline
        muted={muted}
        preload="none"
        onClick={togglePlay}
      />
      <div className="stasharr-reel-info">
        <Link to={`/scenes/${reel.scene.id}`} className="stasharr-reel-title">
          {sceneTitle(reel)}
        </Link>
        {reel.scene.studio && (
          <div className="stasharr-reel-studio">{reel.scene.studio.name}</div>
        )}
        {performers && <div className="stasharr-reel-performers">{performers}</div>}
        <div className="stasharr-reel-meta">
          {Math.round(reel.duration)}s · {sourceLabel(reel.source)}
        </div>
      </div>
      <div className="stasharr-reel-actions">
        <Button variant="link" title={muted ? "Unmute" : "Mute"} onClick={onToggleMute}>
          <Icon icon={muted ? faVolumeMute : faVolumeUp} />
        </Button>
        {canEdit && (
          <>
            <Button variant="link" title="Make again" onClick={() => onRemake(reel)}>
              <Icon icon={faRedo} />
            </Button>
            <Button variant="link" title="Delete reel" onClick={() => onDelete(reel)}>
              <Icon icon={faTrashAlt} />
            </Button>
          </>
        )}
      </div>
    </div>
  );
};

export const Reels: React.FC = () => {
  const Toast = useToast();
  const { isEditor, canViewReels } = useCurrentUser();
  const [page, setPage] = useState(1);
  const [reels, setReels] = useState<Reel[]>([]);
  const [total, setTotal] = useState(0);
  const [activeID, setActiveID] = useState<string>();
  const [muted, setMuted] = useState(true);
  const [shuffled, setShuffled] = useState(false);
  const itemRefs = useRef(new Map<string, HTMLDivElement>());
  const observer = useRef<IntersectionObserver>();

  const { data, loading, error } = GQL.useFindReelsQuery({
    variables: { page, per_page: perPage },
    fetchPolicy: "network-only",
    skip: !canViewReels,
  });
  const [destroyReel] = GQL.useReelDestroyMutation();
  const [generateReels] = GQL.useGenerateReelsMutation();

  useEffect(() => {
    if (!data) return;
    const found = data.findReels;
    setTotal(found.count);
    setReels((prev) => {
      const seen = new Set(prev.map((r) => r.id));
      const fresh = found.reels.filter((r) => !seen.has(r.id));
      return page === 1 ? found.reels : prev.concat(fresh);
    });
  }, [data, page]);

  // play the reel that is mostly on screen
  useEffect(() => {
    observer.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting && e.intersectionRatio >= 0.6) {
            const id = (e.target as HTMLElement).dataset.reelId;
            if (id) setActiveID(id);
          }
        });
      },
      { threshold: [0.6] }
    );
    itemRefs.current.forEach((el) => observer.current?.observe(el));
    return () => observer.current?.disconnect();
  }, []);

  const setItemRef = useCallback((id: string, el: HTMLDivElement | null) => {
    const existing = itemRefs.current.get(id);
    if (existing && existing !== el) {
      observer.current?.unobserve(existing);
      itemRefs.current.delete(id);
    }
    if (el) {
      itemRefs.current.set(id, el);
      observer.current?.observe(el);
    }
  }, []);

  // load more when the last few reels come into view
  useEffect(() => {
    if (!activeID || loading) return;
    const idx = reels.findIndex((r) => r.id === activeID);
    if (idx >= reels.length - 3 && reels.length < total) {
      setPage((p) => p + 1);
    }
  }, [activeID, reels, total, loading]);

  useEffect(() => {
    if (!activeID && reels.length > 0) {
      setActiveID(reels[0].id);
    }
  }, [activeID, reels]);

  async function onDelete(reel: Reel) {
    if (!window.confirm("Delete this reel?")) return;
    try {
      await destroyReel({ variables: { id: reel.id } });
      setReels((rs) => rs.filter((r) => r.id !== reel.id));
      setTotal((t) => t - 1);
    } catch (e) {
      Toast.error(e);
    }
  }

  async function onRemake(reel: Reel) {
    try {
      await generateReels({
        variables: { input: { scene_ids: [reel.scene.id], overwrite: true } },
      });
      Toast.success("Making a new reel. It will appear here when it is done.");
    } catch (e) {
      Toast.error(e);
    }
  }

  function shuffle() {
    setReels((rs) => {
      const copy = [...rs];
      for (let i = copy.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [copy[i], copy[j]] = [copy[j], copy[i]];
      }
      return copy;
    });
    setShuffled(true);
    setActiveID(undefined);
    window.scrollTo(0, 0);
  }

  if (!canViewReels) {
    return (
      <div className="stasharr-reels-empty">
        <h2>Reels</h2>
        <p>Your tier does not include reels.</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="stasharr-reels-empty">
        <h2>Reels</h2>
        <p>{error.message}</p>
      </div>
    );
  }

  if (loading && reels.length === 0) return <LoadingIndicator />;

  if (reels.length === 0) {
    return (
      <div className="stasharr-reels-empty">
        <Helmet>
          <title>Reels</title>
        </Helmet>
        <h2>No reels yet</h2>
        <p>
          Reels are 15 second vertical clips made from the best moments of a
          scene.
        </p>
        {isEditor && (
          <p>
            Open a scene and choose <strong>Make reel</strong> from its menu,
            or make reels for the whole library under Settings → Auto-import.
          </p>
        )}
      </div>
    );
  }

  return (
    <div className="stasharr-reels">
      <Helmet>
        <title>Reels</title>
      </Helmet>
      <div className="stasharr-reels-toolbar">
        <span>
          {total} reel{total === 1 ? "" : "s"}
        </span>
        <Button
          variant="secondary"
          size="sm"
          title="Shuffle"
          active={shuffled}
          onClick={() => shuffle()}
        >
          <Icon icon={faRandom} /> Shuffle
        </Button>
      </div>
      {reels.map((r) => (
        <ReelItem
          key={r.id}
          reel={r}
          active={r.id === activeID}
          muted={muted}
          canEdit={isEditor}
          onToggleMute={() => setMuted(!muted)}
          onDelete={onDelete}
          onRemake={onRemake}
          setRef={(el) => setItemRef(r.id, el)}
        />
      ))}
      {loading && <LoadingIndicator inline small />}
    </div>
  );
};

export default Reels;
