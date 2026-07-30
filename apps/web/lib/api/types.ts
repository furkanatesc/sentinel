import type { RiskLevel } from "@/lib/format";
import type { ScoreKey } from "@/lib/token/score-defs";
import type { RiskSeverity } from "@/lib/format";

export interface Kpi {
  id: string;
  label: string;
  value: string;
  change: number;
  spark: number[];
  updated: string;
  tone?: "positive" | "critical" | "warning" | "neutral";
}

export interface TokenRow {
  id: string;
  name: string;
  symbol: string;
  mint: string;
  ageSeconds: number;
  price: number;
  liquidity: number;
  vol5m: number;
  holders: number;
  creatorScore: number;
  safetyScore: number;
  momentum: number;
  spark: number[];
  signal: "buy" | "watch" | "avoid" | null;
  watchlisted: boolean;
}

export interface AlertEvent {
  id: string;
  type: string;
  token: string;
  detail: string;
  severity: import("@/lib/format").AlertSeverity;
  time: string;
}

export interface RadarPoint {
  x: number;
  y: number;
  z: number;
  name: string;
  level: RiskLevel;
}

export interface ScoreBreakdownItem { label: string; weight: number; detail: string; }
export interface ScoreDetail { key: ScoreKey; value: number; confidence: number; updatedAt: string; breakdown: ScoreBreakdownItem[]; }
export interface RiskItem { id: string; title: string; severity: RiskSeverity; description: string; evidence?: string; firstSeen: string; lastSeen: string; }
export interface RiskGroups { contract: RiskItem[]; market: RiskItem[]; creator: RiskItem[]; }
export interface SeriesPoint { t: number; v: number; }
export interface TokenMetrics {
  holders: number; uniqueBuyers: number; buyRatio: number; sellRatio: number;
  creatorHoldingPct: number; top10HolderPct: number; sniperPct: number; botActivityPct: number;
}
export interface TokenDetail {
  id: string; name: string; symbol: string; mint: string;
  ageSeconds: number; price: number; priceChange24h: number;
  marketCap: number; liquidity: number; volume24h: number;
  scores: Record<ScoreKey, ScoreDetail>;
  metrics: TokenMetrics;
  series: { price: SeriesPoint[]; liquidity: SeriesPoint[]; volume: SeriesPoint[]; holders: SeriesPoint[] };
  risks: RiskGroups;
}
