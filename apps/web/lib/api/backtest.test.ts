import { mockApi } from "./mock";
import type { BacktestParams } from "./types";

const base: BacktestParams = {
  strategyId: "momentum-scalp", rangePreset: "30g", initialCapitalSol: 100, maxPositions: 5,
  slippageModel: "dynamic", priorityFee: 0.0001, latencyModel: "realistic", liquidityModel: "constrained",
  minCreatorScore: 60, minTokenSafety: 55,
};

test("runBacktest is deterministic for identical params", async () => {
  expect(await mockApi.runBacktest(base)).toEqual(await mockApi.runBacktest(base));
});

test("runBacktest is params-sensitive", async () => {
  const a = await mockApi.runBacktest(base);
  const b = await mockApi.runBacktest({ ...base, strategyId: "safe-graduation", initialCapitalSol: 500 });
  expect(a.metrics.netPnlSol).not.toBe(b.metrics.netPnlSol);
});

test("runBacktest is sensitive to priorityFee alone", async () => {
  const a = await mockApi.runBacktest(base);
  const b = await mockApi.runBacktest({ ...base, priorityFee: base.priorityFee + 0.05 });
  expect(a.metrics.netPnlSol).not.toBe(b.metrics.netPnlSol);
});

test.each([
  ["strategyId", { strategyId: "safe-graduation" }],
  ["rangePreset", { rangePreset: "90g" }],
  ["initialCapitalSol", { initialCapitalSol: 250 }],
  ["maxPositions", { maxPositions: 9 }],
  ["slippageModel", { slippageModel: "pessimistic" }],
  ["priorityFee", { priorityFee: base.priorityFee + 0.05 }],
  ["latencyModel", { latencyModel: "high" }],
  ["liquidityModel", { liquidityModel: "unconstrained" }],
  ["minCreatorScore", { minCreatorScore: 55 }],
  ["minTokenSafety", { minTokenSafety: 60 }],
] as [string, Partial<BacktestParams>][])(
  "runBacktest result differs when %s alone changes",
  async (_field, override) => {
    const baseline = await mockApi.runBacktest(base);
    const changed = await mockApi.runBacktest({ ...base, ...override });
    expect(changed).not.toEqual(baseline);
  },
);

test("runBacktest is sensitive to swapping minCreatorScore <-> minTokenSafety (former seed-collision case)", async () => {
  const baseline = await mockApi.runBacktest(base);
  const swapped = await mockApi.runBacktest({
    ...base,
    minCreatorScore: base.minTokenSafety,
    minTokenSafety: base.minCreatorScore,
  });
  expect(swapped).not.toEqual(baseline);
});

test("runBacktest returns full metrics + series + trades", async () => {
  const r = await mockApi.runBacktest(base);
  expect(r.metrics.trades).toBeGreaterThan(0);
  expect(r.equityCurve.length).toBeGreaterThan(0);
  expect(r.drawdown.every((d) => d.v <= 0)).toBe(true);
  expect(r.monthlyReturns.length).toBeGreaterThan(0);
  expect(r.tradeDistribution.length).toBeGreaterThan(0);
  expect(r.pnlByScore.length).toBeGreaterThan(0);
  expect(r.priceSeries.length).toBeGreaterThan(0);
  const times = new Set(r.priceSeries.map((p) => p.t));
  expect(r.trades.some((t) => t.side === "buy")).toBe(true);
  expect(r.trades.some((t) => t.side === "sell")).toBe(true);
  expect(r.trades.every((t) => times.has(t.time))).toBe(true);
});
