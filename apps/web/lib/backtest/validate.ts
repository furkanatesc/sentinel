import type { BacktestParams } from "@/lib/api/types";

export function validateParams(p: BacktestParams): { [field: string]: string } {
  const e: { [field: string]: string } = {};
  if (!p.strategyId) e.strategyId = "Strateji seç";
  if (!(p.initialCapitalSol > 0)) e.initialCapitalSol = "Sermaye 0'dan büyük olmalı";
  if (!(p.maxPositions >= 1)) e.maxPositions = "En az 1 pozisyon";
  if (p.priorityFee < 0) e.priorityFee = "Öncelik ücreti negatif olamaz";
  if (p.minCreatorScore < 0 || p.minCreatorScore > 100) e.minCreatorScore = "Skor 0–100 arası";
  if (p.minTokenSafety < 0 || p.minTokenSafety > 100) e.minTokenSafety = "Skor 0–100 arası";
  return e;
}
