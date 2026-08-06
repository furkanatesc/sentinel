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

const sampleEvent = [{
  id: "e1", type: "new_mint", symbol: "PULSE", mint: "9xQeWv...4Fk2",
  launchpad: "pump.fun", dex: "raydium", liquidity: 82400, creatorScore: 0,
  riskLevel: "medium", tokenAgeSeconds: 38, volume5m: 41200, holderGrowthPct: 12,
  severity: "info", detail: "yeni mint", time: "2026-08-05T10:00:00Z", ts: 1,
  watchlisted: false,
}];

it("getEvents API JSON'unu FeedEvent[]'e maple", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(JSON.stringify(sampleEvent), { status: 200, headers: { "content-type": "application/json" } })));
  const rows = await httpApi.getEvents();
  expect(rows).toEqual(sampleEvent);
});

const sampleToken = [{
  id: "t1", name: "SolPulse", symbol: "PULSE", mint: "9xQeWv...4Fk2", ageSeconds: 38,
  price: 0.0042, liquidity: 82400, vol5m: 41200, holders: 312, creatorScore: 0, safetyScore: 0,
  momentum: 88, spark: [1, 2, 3], signal: "buy", watchlisted: true,
}];

it("getTokens API JSON'unu TokenRow[]'a maple", async () => {
  vi.stubGlobal("fetch", vi.fn(async () =>
    new Response(JSON.stringify(sampleToken), { status: 200, headers: { "content-type": "application/json" } })));
  const rows = await httpApi.getTokens();
  expect(rows).toEqual(sampleToken);
});

const sampleDetail = {
  id: "t1", name: "SolPulse", symbol: "PULSE", mint: "9xQeWv...4Fk2", ageSeconds: 38,
  price: 0.0042, priceChange24h: 3.2, marketCap: 420000, liquidity: 82400, volume24h: 41200,
  scores: {}, metrics: {}, series: { price: [], liquidity: [], volume: [], holders: [] }, risks: { contract: [], market: [], creator: [] },
};

it("getToken API JSON'unu TokenDetail'e maple ve /api/token/{mint} çağırır", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(JSON.stringify(sampleDetail), { status: 200, headers: { "content-type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const got = await httpApi.getToken("9xQeWv...4Fk2");
  expect(got).toEqual(sampleDetail);
  expect(fetchMock).toHaveBeenCalledWith(
    "https://api.test/api/token/9xQeWv...4Fk2",
    expect.objectContaining({ headers: { accept: "application/json" } }),
  );
});

it("subscribeEvents WS üzerinden events topic'ine abone olur", () => {
  class MockWS {
    static instances: MockWS[] = [];
    onmessage: ((e: { data: string }) => void) | null = null;
    closed = false;
    constructor(public url: string) { MockWS.instances.push(this); }
    close() { this.closed = true; }
  }
  vi.stubGlobal("WebSocket", MockWS as unknown as typeof WebSocket);
  const cb = vi.fn();
  const unsub = httpApi.subscribeEvents(cb);
  const ws = MockWS.instances[0];
  ws.onmessage?.({ data: JSON.stringify({ topic: "events", payload: sampleEvent[0] }) });
  expect(cb).toHaveBeenCalledWith(sampleEvent[0]);
  unsub();
  expect(ws.closed).toBe(true);
});

it("subscribeTokens WS üzerinden tokens topic'ine abone olur (tam liste snapshot'ı)", () => {
  class MockWS {
    static instances: MockWS[] = [];
    onmessage: ((e: { data: string }) => void) | null = null;
    closed = false;
    constructor(public url: string) { MockWS.instances.push(this); }
    close() { this.closed = true; }
  }
  vi.stubGlobal("WebSocket", MockWS as unknown as typeof WebSocket);
  const cb = vi.fn();
  const unsub = httpApi.subscribeTokens(cb);
  const ws = MockWS.instances[0];
  ws.onmessage?.({ data: JSON.stringify({ topic: "tokens", payload: sampleToken }) });
  expect(cb).toHaveBeenCalledWith(sampleToken);
  unsub();
  expect(ws.closed).toBe(true);
});
