import videojs, { VideoJsPlayer } from "video.js";

// Stasharr: back and forward buttons beside the play button. They draw their
// own icons, so they show whatever the icon font does, and they skip the
// set number of seconds.

interface ISeekOptions extends videojs.ComponentOptions {
  direction: "back" | "forward";
  seconds: number;
}

// a circular arrow with the number of seconds in it
function icon(direction: "back" | "forward", seconds: number): string {
  const flip = direction === "forward" ? ' transform="matrix(-1 0 0 1 24 0)"' : "";
  return (
    `<svg class="vjs-stasharr-seek-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">` +
    `<g${flip} fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round">` +
    `<path d="M4.5 12a7.5 7.5 0 1 0 2.2-5.3"/><path d="M4.3 3.6v3.6h3.6"/></g>` +
    `<text x="12.6" y="15.6" text-anchor="middle" font-size="7.4" font-weight="700" fill="currentColor" font-family="Arial, sans-serif">${seconds}</text>` +
    `</svg>`
  );
}

class StasharrSeekButton extends videojs.getComponent("Button") {
  private direction: "back" | "forward";
  private seconds: number;

  constructor(player: VideoJsPlayer, options: ISeekOptions) {
    super(player, options);
    this.direction = options.direction;
    this.seconds = options.seconds;
    const placeholder = this.el().querySelector(".vjs-icon-placeholder");
    if (placeholder) placeholder.innerHTML = icon(this.direction, this.seconds);
    this.controlText(
      this.direction === "back" ? `Back ${this.seconds} seconds` : `Forward ${this.seconds} seconds`
    );
  }

  buildCSSClass() {
    return `vjs-stasharr-seek ${super.buildCSSClass()}`;
  }

  handleClick(event: Event) {
    event.stopPropagation();
    const p = this.player();
    const now = p.currentTime() ?? 0;
    const end = p.duration();
    let to = now + (this.direction === "back" ? -this.seconds : this.seconds);
    if (Number.isFinite(end) && end > 0) to = Math.min(to, end - 0.5);
    p.currentTime(Math.max(0, to));
    p.userActive(true);
  }
}

class Seek10Plugin extends videojs.getPlugin("plugin") {
  constructor(player: VideoJsPlayer, options?: { seconds?: number }) {
    super(player);
    const seconds = options?.seconds ?? 10;
    player.ready(() => {
      const bar = player.controlBar;
      const play = bar.getChild("playToggle");
      const back = new StasharrSeekButton(player, { direction: "back", seconds });
      const forward = new StasharrSeekButton(player, { direction: "forward", seconds });
      bar.addChild(back);
      bar.addChild(forward);
      // put them right after the play button
      const after = play?.el().nextSibling ?? null;
      bar.el().insertBefore(back.el(), after);
      bar.el().insertBefore(forward.el(), back.el().nextSibling);
    });
  }
}

videojs.registerComponent("StasharrSeekButton", StasharrSeekButton);
videojs.registerPlugin("seek10", Seek10Plugin);

declare module "video.js" {
  interface VideoJsPlayer {
    seek10: () => Seek10Plugin;
  }
  interface VideoJsPlayerPluginOptions {
    seek10?: { seconds?: number };
  }
}

export default Seek10Plugin;
