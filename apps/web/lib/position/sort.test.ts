import type { Position } from "@/lib/api/types";
import { sortPositions } from "./sort";

function pos(overrides: Partial<Position>): Position {
  return {
    id: "id", tokenMint: "mint", tokenSymbol: "SYM",
    strategyId: "s1", strategyName: "Strategy",
    entryPrice: 1, currentPrice: 1, sizeSol: 1,
    pnlSol: 0, pnlPct: 0,
    stopLossPct: 10, takeProfitPct: 50,
    tokenRisk: "good", creatorRisk: "good",
    ageLabel: "1 dk", openedAt: "1 dk önce",
    ...overrides,
  };
}

test("sorts by pnlSol descending (numeric)", () => {
  const rows = [pos({ id: "a", pnlSol: -5 }), pos({ id: "b", pnlSol: 20 }), pos({ id: "c", pnlSol: 3 })];
  const sorted = sortPositions(rows, "pnlSol");
  expect(sorted.map((r) => r.id)).toEqual(["b", "c", "a"]);
});

test("sorts by sizeSol descending (numeric)", () => {
  const rows = [pos({ id: "a", sizeSol: 4 }), pos({ id: "b", sizeSol: 30 }), pos({ id: "c", sizeSol: 12 })];
  const sorted = sortPositions(rows, "sizeSol");
  expect(sorted.map((r) => r.id)).toEqual(["b", "c", "a"]);
});

test("sorts ageLabel by parseInt, not lexicographically", () => {
  // Lexicographic order would put "8 dk" after "12 dk" alphabetically ("1" < "8"),
  // but numerically 12 > 8, so "12 dk" must come first.
  const rows = [pos({ id: "young", ageLabel: "8 dk" }), pos({ id: "old", ageLabel: "12 dk" })];
  const sorted = sortPositions(rows, "ageLabel");
  expect(sorted.map((r) => r.id)).toEqual(["old", "young"]);
});

test("does not mutate the input array", () => {
  const rows = [pos({ id: "a", pnlSol: -5 }), pos({ id: "b", pnlSol: 20 })];
  const original = [...rows];
  sortPositions(rows, "pnlSol");
  expect(rows).toEqual(original);
});
