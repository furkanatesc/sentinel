import type { RiskLevel } from "@/lib/format";

export const POSITION_RISK_LEVELS: RiskLevel[] = ["strong", "good", "medium", "high", "critical"];

export function pnlColor(v: number): string {
  return v >= 0 ? "#2FD98B" : "#F0476B";
}
