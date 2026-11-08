import React, { useEffect, useRef, useState } from "react";

// A short silent loop (like a tag's clip) that only loads and plays while it
// is on screen, so a page full of them stays light. People who ask their
// system for less motion get the still picture.
export const LoopClip: React.FC<{
  video: string;
  still?: string | null;
  className?: string;
  alt?: string;
}> = ({ video, still, className, alt = "" }) => {
  const ref = useRef<HTMLVideoElement>(null);
  const [seen, setSeen] = useState(false);
  const [failed, setFailed] = useState(false);
  const calm =
    typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;

  useEffect(() => {
    const v = ref.current;
    if (!v || calm || failed) return;
    if (typeof IntersectionObserver === "undefined") {
      setSeen(true);
      return;
    }
    const io = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setSeen(true);
          v.play().catch(() => {});
        } else {
          v.pause();
        }
      },
      { threshold: 0.4 }
    );
    io.observe(v);
    return () => io.disconnect();
  }, [calm, failed]);

  if (calm || failed) {
    return still ? <img className={className} src={still} alt={alt} loading="lazy" /> : null;
  }
  return (
    <video
      ref={ref}
      className={className}
      src={seen ? video : undefined}
      poster={still ?? undefined}
      muted
      loop
      playsInline
      autoPlay={seen}
      preload="none"
      aria-label={alt || undefined}
      onError={() => setFailed(true)}
    />
  );
};
