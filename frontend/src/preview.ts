import type { Part } from "./api";

// What each layer looks like before it's recoloured (Tekken's own blue-white).
export const NATIVE: Record<Part["role"], string> = {
  core: "#eaf3ff",
  glow: "#8fb6ff",
  outer: "#b4a6ff",
  trail: "#6f86ff",
};
