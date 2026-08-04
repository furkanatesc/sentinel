import { beforeEach, afterEach, it, expect, vi } from "vitest";
import { httpApi } from "./http";

const OLD = process.env.NEXT_PUBLIC_API_BASE_URL;
beforeEach(() => { process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.test"; });
afterEach(() => {
  process.env.NEXT_PUBLIC_API_BASE_URL = OLD;
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

const sample = [{
  id: "momentum-scalp", name: "Momentum Scalp", status: "live", timeframe: "1-5 dk",
  winRatePct: 63, profitFactor: 1.8, maxDrawdownPct: 16, totalTrades: 298, netPnlSol: 338, lastSignal: "43 dk önce",
}];

it("getStrategies API JSON'unu StrategyRow[]'a maple", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(JSON.stringify(sample), { status: 200, headers: { "content-type": "application/json" } })));
  const rows = await httpApi.getStrategies();
  expect(rows).toEqual(sample);
});

it("getStrategies non-200'de reject eder", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response("boom", { status: 500 })));
  await expect(httpApi.getStrategies()).rejects.toThrow(/500/);
});

it("getStrategies API base yoksa reject eder", async () => {
  delete process.env.NEXT_PUBLIC_API_BASE_URL;
  await expect(httpApi.getStrategies()).rejects.toThrow(/NEXT_PUBLIC_API_BASE_URL/);
});
