import { mockApi } from "./mock";

test("getCandles is deterministic and returns OHLC series for a known mint", async () => {
  const a = await mockApi.getCandles("9xQeWv...4Fk2");
  const b = await mockApi.getCandles("9xQeWv...4Fk2");
  expect(a).toEqual(b);
  expect(a.length).toBeGreaterThanOrEqual(30);
  for (const c of a) {
    expect(c.high).toBeGreaterThanOrEqual(c.low);
    expect(typeof c.time).toBe("number");
  }
});

test("getMarketData resolves a token symbol/mint to price + scores", async () => {
  const m = await mockApi.getMarketData("PULSE");
  expect(m.symbol).toBe("PULSE");
  expect(m.price).toBeGreaterThan(0);
  expect(m.tokenScore).toBeGreaterThanOrEqual(0);
  expect(m.creatorScore).toBeGreaterThanOrEqual(0);
});

test("orders/transactions/logs are non-empty and deterministic", async () => {
  expect(await mockApi.getOrders()).toEqual(await mockApi.getOrders());
  expect((await mockApi.getOrders()).length).toBeGreaterThan(0);
  expect((await mockApi.getTransactions()).length).toBeGreaterThan(0);
  const logs = await mockApi.getTradeLogs();
  expect(logs.length).toBeGreaterThan(0);
  expect(["info", "warn", "error"]).toContain(logs[0].level);
});
