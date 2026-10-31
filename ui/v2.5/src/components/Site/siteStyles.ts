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
