import type { CreatorOutcome, LiquidityStatus } from "@/lib/api/types";

export const OUTCOME_DEFS: Record<CreatorOutcome, { label: string; color: string }> = {
  active: { label: "Aktif", color: "#3E9BFF" },
  graduated: { label: "Graduated", color: "#2FD98B" },
  dumped: { label: "Dump", color: "#FFB020" },
  rug: { label: "Rug", color: "#F0476B" },
  dead: { label: "Ölü", color: "#8A94A6" },
};

export const LIQUIDITY_DEFS: Record<LiquidityStatus, { label: string; color: string }> = {
  locked: { label: "Kilitli", color: "#2FD98B" },
  unlocked: { label: "Kilitsiz", color: "#FFB020" },
  removed: { label: "Çekildi", color: "#F0476B" },
};
