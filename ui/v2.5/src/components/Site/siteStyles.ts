// The looks an admin can pick for the site front end. Each style is a set
// of CSS variables (see site.scss) plus a few layout choices.

export interface ISiteStyle {
  id: string;
  name: string;
  description: string;
  // full: one big picture across the page; split: big picture with the next
  // picks stacked beside it
  hero: "full" | "split";
  // colours shown on the picker
  swatches: string[];
}

export const siteStyles: ISiteStyle[] = [
  {
    id: "velvet",
    name: "Velvet",
    description:
      "Black and gold with elegant serif titles. Feels like a luxury studio site.",
    hero: "full",
    swatches: ["#0b0a09", "#1a1714", "#c9a45c", "#f3e7cf"],
  },
  {
    id: "neon",
    name: "Neon Nights",
    description:
      "Deep purple with glowing pink and cyan. Bold, playful and modern.",
    hero: "split",
    swatches: ["#0d0718", "#1c1030", "#ff2e93", "#21e6ff"],
  },
  {
    id: "crimson",
    name: "Crimson",
    description:
      "Charcoal and red with heavy uppercase headlines. Loud and punchy.",
    hero: "full",
    swatches: ["#111111", "#1d1d1d", "#e10600", "#ffffff"],
  },
  {
    id: "blush",
    name: "Blush",
    description:
      "Light and airy: white, soft pink and rounded cards, like a glossy magazine.",
    hero: "split",
    swatches: ["#fff7f9", "#ffffff", "#e84a7f", "#2b1d24"],
  },
  {
    id: "onyx",
    name: "Onyx",
    description:
      "Minimal monochrome with big pictures and thin lines. Clean and editorial.",
    hero: "full",
    swatches: ["#000000", "#121212", "#ffffff", "#8a8a8a"],
  },
];

export function findSiteStyle(id: string | undefined): ISiteStyle {
  return siteStyles.find((s) => s.id === id) ?? siteStyles[0];
}

// Where things sit on the pages. Any layout works with any style.
export type SiteNavPlace = "top" | "dock";

export interface ISiteLayout {
  id: string;
  name: string;
  description: string;
  nav: SiteNavPlace;
  // a rough sketch for the picker: boxes as [left, top, width, height] in %
  sketch: [number, number, number, number][];
}

export const siteLayouts: ISiteLayout[] = [
  {
    id: "classic",
    name: "Classic",
    description: "Menu across the top, a big banner, then rows of scenes. The original.",
    nav: "top",
    sketch: [
      [0, 0, 100, 8],
      [0, 8, 100, 40],
      [4, 54, 28, 18],
      [36, 54, 28, 18],
      [68, 54, 28, 18],
      [4, 78, 28, 18],
      [36, 78, 28, 18],
      [68, 78, 28, 18],
    ],
  },
  {
    id: "mosaic",
    name: "Mosaic",
    description:
      "A wide search bar, a sticky row of category chips and a masonry wall of mixed-shape scene and model cards that play when pointed at.",
    nav: "top",
    sketch: [
      [0, 0, 100, 8],
      [4, 12, 60, 6],
      [4, 22, 22, 38],
      [28, 22, 22, 24],
      [52, 22, 22, 44],
      [76, 22, 20, 30],
      [4, 63, 22, 34],
      [28, 49, 22, 48],
      [52, 69, 22, 28],
      [76, 55, 20, 42],
    ],
  },
  {
    id: "bento",
    name: "Bento",
    description:
      "A floating glass menu bar and a home page of mixed-size tiles: the featured scene, trending picks, the star of the week, categories and new arrivals.",
    nav: "top",
    sketch: [
      [20, 2, 60, 8],
      [4, 14, 46, 52],
      [52, 14, 21, 34],
      [75, 14, 21, 34],
      [52, 50, 21, 34],
      [75, 50, 21, 34],
      [4, 68, 46, 16],
      [4, 88, 92, 10],
    ],
  },
  {
    id: "showcase",
    name: "Showcase",
    description:
      "A 3D cover-flow of the featured scenes over a soft glow of the picture, with its details centred below, then the rows.",
    nav: "top",
    sketch: [
      [0, 0, 100, 8],
      [4, 18, 14, 30],
      [82, 18, 14, 30],
      [14, 14, 16, 38],
      [70, 14, 16, 38],
      [24, 12, 52, 42],
      [30, 58, 40, 8],
      [4, 74, 28, 20],
      [36, 74, 28, 20],
      [68, 74, 28, 20],
    ],
  },
  {
    id: "spotlight",
    name: "Spotlight",
    description:
      "A split screen: the chosen scene plays big on the left while a numbered top-10 list scrolls on the right.",
    nav: "top",
    sketch: [
      [0, 0, 100, 7],
      [0, 7, 56, 93],
      [60, 12, 36, 8],
      [60, 24, 36, 14],
      [60, 42, 36, 14],
      [60, 60, 36, 14],
      [60, 78, 36, 14],
    ],
  },
  {
    id: "immersive",
    name: "Immersive",
    description:
      "The featured scene fills the whole screen, the rows sit on a frosted sheet over it, and the menu is a floating dock at the bottom.",
    nav: "dock",
    sketch: [
      [0, 0, 100, 62],
      [0, 50, 100, 50],
      [4, 58, 28, 16],
      [36, 58, 28, 16],
      [68, 58, 28, 16],
      [30, 86, 40, 9],
    ],
  },
  {
    id: "editorial",
    name: "Editorial",
    description:
      "A glossy magazine: a masthead, a cover story, an \u201cIn this issue\u201d list and features, serif headlines and black-and-white portraits that turn colour when pointed at.",
    nav: "top",
    sketch: [
      [30, 2, 40, 7],
      [4, 11, 92, 4],
      [4, 20, 56, 44],
      [64, 24, 32, 6],
      [64, 34, 28, 3],
      [64, 40, 30, 3],
      [4, 70, 28, 3],
      [36, 70, 28, 3],
      [68, 70, 28, 3],
      [4, 80, 50, 17],
      [58, 80, 38, 17],
    ],
  },
  {
    id: "brutal",
    name: "Brutal",
    description:
      "Bold and loud: thick borders, hard shadows, giant capitals, stickers and a scrolling marquee. Cards jump when pointed at and press in when clicked.",
    nav: "top",
    sketch: [
      [0, 0, 100, 9],
      [4, 14, 46, 46],
      [54, 16, 42, 42],
      [0, 64, 100, 8],
      [4, 78, 28, 18],
      [36, 78, 28, 18],
      [68, 78, 28, 18],
    ],
  },
  {
    id: "swipe",
    name: "Swipe",
    description:
      "Discover by swiping: a deck of scenes you drag right to like or left to skip, with soft rounded cards and a floating action bar on every scene.",
    nav: "top",
    sketch: [
      [0, 0, 100, 8],
      [4, 18, 22, 20],
      [36, 16, 28, 54],
      [33, 20, 34, 50],
      [74, 18, 22, 40],
      [38, 74, 24, 8],
      [4, 86, 92, 12],
    ],
  },
];

export function findSiteLayout(id: string | null | undefined): ISiteLayout {
  return siteLayouts.find((l) => l.id === id) ?? siteLayouts[0];
}

// A viewer's own choice of layout, kept in this browser only.
const LAYOUT_KEY = "stasharr-site-layout";

export function viewerLayout(): string | null {
  try {
    const v = localStorage.getItem(LAYOUT_KEY);
    return v && siteLayouts.some((l) => l.id === v) ? v : null;
  } catch {
    return null;
  }
}

export function setViewerLayout(id: string | null) {
  try {
    if (id) localStorage.setItem(LAYOUT_KEY, id);
    else localStorage.removeItem(LAYOUT_KEY);
  } catch {
    // private window: not remembered
  }
}
