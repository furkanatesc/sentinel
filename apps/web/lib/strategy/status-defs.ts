import type { StrategyStatus, StrategyCondition } from "@/lib/api/types";

export const STATUS_DEFS: Record<StrategyStatus, { label: string; color: string }> = {
  draft: { label: "Taslak", color: "#8A94A6" },
  backtesting: { label: "Backtest", color: "#3E9BFF" },
  paper: { label: "Kâğıt", color: "#7C5CFF" },
  shadow: { label: "Gölge", color: "#FFB020" },
  live: { label: "Canlı", color: "#2FD98B" },
  paused: { label: "Duraklatıldı", color: "#F0476B" },
  archived: { label: "Arşiv", color: "#5A6474" },
};

export const CONDITION_LABELS: Record<string, string> = {
  creatorScore: "Creator Skoru",
  tokenSafety: "Token Güvenliği",
  liquidity: "Likidite",
  holderGrowth5m: "Holder Büyümesi 5dk",
  manipulationRisk: "Manipülasyon Riski",
  momentum: "Momentum",
  ageSeconds: "Yaş",
};

export function formatCondition(c: StrategyCondition): string {
  const label = CONDITION_LABELS[c.metric] ?? c.metric;
  const unit = c.unit ? ` ${c.unit}` : "";
  return `${label} ${c.op} ${c.value}${unit}`;
}
