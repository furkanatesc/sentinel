import type { RiskLevel } from "@/lib/format";

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
