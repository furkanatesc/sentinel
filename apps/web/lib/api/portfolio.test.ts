import { mockApi } from "./mock";

test("getPortfolio returns deterministic summary + all four series", async () => {
  const a = await mockApi.getPortfolio();
  const b = await mockApi.getPortfolio();
  expect(a).toEqual(b);
  expect(typeof a.summary.totalValueSol).toBe("number");
  expect(a.equityCurve.length).toBeGreaterThan(1);
  expect(a.pnlByStrategy.length).toBeGreaterThan(0);
  expect(a.riskAllocation.length).toBeGreaterThan(0);
  expect(a.riskAllocation.every((s) => s.color.startsWith("#"))).toBe(true);
  expect(a.winLoss.length).toBeGreaterThan(0);
});

test("getPositions returns deterministic rows referencing real token symbols + strategy ids", async () => {
  const a = await mockApi.getPositions();
  const b = await mockApi.getPositions();
  expect(a).toEqual(b);
  expect(a.length).toBeGreaterThanOrEqual(6);
  for (const p of a) {
    expect(p.tokenSymbol).toBeTruthy();
    expect(p.strategyId).toBeTruthy();
    expect(["critical", "high", "medium", "good", "strong"]).toContain(p.tokenRisk);
    expect(typeof p.pnlSol).toBe("number");
  }
});
