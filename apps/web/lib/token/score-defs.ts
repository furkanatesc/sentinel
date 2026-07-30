import { scoreToLevel, type RiskLevel } from "@/lib/format";

export type ScoreKey = "opportunity" | "creatorReputation" | "tokenSafety" | "manipulationRisk";

export interface ScoreDef {
  key: ScoreKey;
  label: string;
  higherIsBetter: boolean;
}

export const SCORE_DEFS: ScoreDef[] = [
  { key: "opportunity", label: "Fırsat Skoru", higherIsBetter: true },
  { key: "creatorReputation", label: "Üretici İtibarı", higherIsBetter: true },
  { key: "tokenSafety", label: "Token Güvenliği", higherIsBetter: true },
  { key: "manipulationRisk", label: "Manipülasyon Riski", higherIsBetter: false },
];

/** Renk seviyesi: yüksek-kötü skorlarda ters çevrilir (100 - value). */
export function scoreDisplayLevel(value: number, higherIsBetter: boolean): RiskLevel {
  return scoreToLevel(higherIsBetter ? value : 100 - value);
}
