// The mini-player: leave a scene while it plays and it keeps going in a
// small window in the corner while you browse.

import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { useHistory, useLocation } from "react-router-dom";
import { ScenePlayer } from "src/components/ScenePlayer/ScenePlayer";
import { t } from "./i18n";

type PlayerScene = React.ComponentProps<typeof ScenePlayer>["scene"];

interface IMini {
  scene: PlayerScene;
  time: number;
}

interface IMiniContext {
  mini: IMini | null;
  open: (m: IMini) => void;
  close: () => void;
}

const MiniContext = createContext<IMiniContext>({ mini: null, open: () => {}, close: () => {} });

export const MiniPlayerProvider: React.FC = ({ children }) => {
  const [mini, setMini] = useState<IMini | null>(null);
  return (
    <MiniContext.Provider value={{ mini, open: setMini, close: () => setMini(null) }}>
      {children}
    </MiniContext.Provider>
  );
};

export const useMiniPlayer = () => useContext(MiniContext);

const OFF_KEY = "stasharr-mini-player-off";

export function miniPlayerOn(): boolean {
  try {
    return localStorage.getItem(OFF_KEY) !== "1";
  } catch {
    return true;
  }
}

export function setMiniPlayerOn(on: boolean) {
  try {
    if (on) localStorage.removeItem(OFF_KEY);
    else localStorage.setItem(OFF_KEY, "1");
  } catch {
    // ignore
  }
}

// Used by the scene page: watches its player, and when you leave the page
// while it plays, hands the scene to the mini-player at the same moment.
export function useHandOffToMini(scene: PlayerScene | undefined, selector = ".site-player video") {
  const { open, close } = useMiniPlayer();
  const last = useRef<{ time: number; playing: boolean }>({ time: 0, playing: false });

  // a scene page plays by itself, so the mini-player closes
  useEffect(() => {
    close();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scene?.id]);

  useEffect(() => {
    if (!scene) return;
    last.current = { time: 0, playing: false };
    const timer = window.setInterval(() => {
      const v = document.querySelector<HTMLVideoElement>(selector);
      if (v) last.current = { time: v.currentTime, playing: !v.paused && !v.ended };
    }, 500);
    return () => {
      window.clearInterval(timer);
      const { time, playing } = last.current;
      if (playing && time > 3 && miniPlayerOn()) open({ scene, time });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scene?.id]);
}

type Corner = "br" | "bl" | "tr" | "tl";
const CORNER_KEY = "stasharr-mini-corner";

function savedCorner(): Corner {
  try {
    const c = localStorage.getItem(CORNER_KEY);
    return c === "bl" || c === "tr" || c === "tl" ? c : "br";
  } catch {
    return "br";
  }
}

export const MiniPlayer: React.FC = () => {
  const { mini, close } = useMiniPlayer();
  const history = useHistory();
  const location = useLocation();
  const box = useRef<HTMLDivElement>(null);
  const [corner, setCorner] = useState<Corner>(savedCorner);
  const [drag, setDrag] = useState<{ x: number; y: number } | null>(null);
  const start = useRef<{ x: number; y: number; left: number; top: number } | null>(null);

  // the scene page takes over its own scene
  const onScenePage = mini ? location.pathname === `/site/scene/${mini.scene.id}` : false;
  useEffect(() => {
    if (onScenePage) close();
  }, [onScenePage, close]);

  if (!mini || onScenePage) return null;

  const now = () => box.current?.querySelector("video")?.currentTime ?? mini.time;

  const expand = () => {
    const at = Math.floor(now());
    close();
    history.push(`/site/scene/${mini.scene.id}?t=${at}`);
  };

  const title = mini.scene.title || t("Untitled scene");

  return (
    <div
      ref={box}
      className={`site-mini site-mini-${corner} ${drag ? "dragging" : ""}`}
      style={drag ? { left: drag.x, top: drag.y, right: "auto", bottom: "auto" } : undefined}
      role="region"
      aria-label={t("Mini-player")}
    >
      <div
        className="site-mini-bar"
        onPointerDown={(e) => {
          if ((e.target as HTMLElement).closest("button")) return;
          const r = box.current?.getBoundingClientRect();
          if (!r) return;
          start.current = { x: e.clientX, y: e.clientY, left: r.left, top: r.top };
          e.currentTarget.setPointerCapture(e.pointerId);
        }}
        onPointerMove={(e) => {
          if (!start.current) return;
          setDrag({ x: start.current.left + e.clientX - start.current.x, y: start.current.top + e.clientY - start.current.y });
        }}
        onPointerUp={(e) => {
          if (!start.current) return;
          start.current = null;
          // snap to the nearest corner
          const c: Corner = `${e.clientY < window.innerHeight / 2 ? "t" : "b"}${e.clientX < window.innerWidth / 2 ? "l" : "r"}` as Corner;
          setCorner(c);
          setDrag(null);
          try {
            localStorage.setItem(CORNER_KEY, c);
          } catch {
            // ignore
          }
        }}
      >
        <span className="site-mini-title" title={title}>
          {title}
        </span>
        <button type="button" onClick={expand} aria-label={t("Back to the scene")} title={t("Back to the scene")}>
          ⤢
        </button>
        <button type="button" onClick={close} aria-label={t("Close")} title={t("Close")}>
          ✕
        </button>
      </div>
      <div className="site-mini-video">
        <ScenePlayer
          key={mini.scene.id}
          scene={mini.scene}
          hideScrubberOverride
          autoplay
          permitLoop={false}
          initialTimestamp={mini.time}
          sendSetTimestamp={() => {}}
          onComplete={close}
          onNext={() => {}}
          onPrevious={() => {}}
        />
      </div>
    </div>
  );
};
