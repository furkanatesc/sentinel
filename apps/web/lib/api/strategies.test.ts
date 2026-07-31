import { mockApi } from "./mock";

test("getStrategies returns deterministic rows with required fields", async () => {
  const a = await mockApi.getStrategies();
  const b = await mockApi.getStrategies();
  expect(a.length).toBeGreaterThanOrEqual(5);
  expect(a).toEqual(b); // deterministic
  for (const r of a) {
    expect(r.id).toBeTruthy();
    expect(r.name).toBeTruthy();
    expect(typeof r.winRatePct).toBe("number");
    expect(typeof r.netPnlSol).toBe("number");
  }
});

test("getStrategy returns a fully populated detail for a known id", async () => {
  const rows = await mockApi.getStrategies();
  const d = await mockApi.getStrategy(rows[0].id);
  expect(d.id).toBe(rows[0].id);
  expect(d.entry.length).toBeGreaterThan(0);
  expect(d.exit.length).toBeGreaterThan(0);
  expect(d.equityCurve.length).toBeGreaterThan(1);
  expect(d.performance.sharpe).toBeDefined();
  expect(d.backtest.trades).toBeGreaterThan(0);
  expect(d.versions.length).toBeGreaterThan(0);
  expect(d.audit.length).toBeGreaterThan(0);
});

test("getStrategy falls back to a valid detail for an unknown id", async () => {
  const d = await mockApi.getStrategy("does-not-exist");
  expect(d.name).toBeTruthy();
  expect(d.entry.length).toBeGreaterThan(0);
});
