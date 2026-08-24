import { beforeEach, afterEach, it, expect, vi } from "vitest";
import { httpApi } from "./http";
import { LIVE_ENDPOINTS } from "./live-endpoints";

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

const sampleCreators = [{
  address: "AAA", reputationScore: 0, riskLevel: "medium", totalTokens: 2,
  activeTokens: 0, ruggedTokens: 0, successRatePct: 0, realizedPnlSol: 0,
}];

it("getCreators API JSON'unu CreatorRow[]'a maple ve /api/creators çağırır", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(JSON.stringify(sampleCreators), { status: 200, headers: { "content-type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const rows = await httpApi.getCreators();
  expect(rows).toEqual(sampleCreators);
  expect(fetchMock).toHaveBeenCalledWith(
    "https://api.test/api/creators",
    expect.objectContaining({ headers: { accept: "application/json" } }),
  );
});

const sampleProfile = {
  address: "AAA", walletAgeDays: 0, firstSeen: "2026-08-10T00:00:00Z",
  reputation: { key: "creatorReputation", value: 0, confidence: 0, updatedAt: "", breakdown: [] },
  riskLevel: "medium", metrics: { totalTokens: 2 }, history: [], behavior: { repeatedFunders: [] },
};

it("getCreator API JSON'unu CreatorProfile'e maple ve /api/creator/{address} çağırır", async () => {
  const fetchMock = vi.fn(async () =>
    new Response(JSON.stringify(sampleProfile), { status: 200, headers: { "content-type": "application/json" } }));
  vi.stubGlobal("fetch", fetchMock);
  const got = await httpApi.getCreator("AAA");
  expect(got).toEqual(sampleProfile);
  expect(fetchMock).toHaveBeenCalledWith(
    "https://api.test/api/creator/AAA",
    expect.objectContaining({ headers: { accept: "application/json" } }),
  );
});

it("LIVE_ENDPOINTS getCreators ve getCreator içerir", () => {
  expect(LIVE_ENDPOINTS.has("getCreators")).toBe(true);
  expect(LIVE_ENDPOINTS.has("getCreator")).toBe(true);
});

const sampleKpis = [{ id: "detected", label: "x", value: "3", change: 0, spark: [], updated: "n" }];

it("getKpis gerçek API'den Kpi[] döndürür", async () => {
  vi.spyOn(global, "fetch").mockResolvedValue(new Response(JSON.stringify(sampleKpis)));
  const r = await httpApi.getKpis();
  expect(r[0].id).toBe("detected");
});

const sampleRadar = [{ x: 1, y: 2, z: 3, name: "A", level: "good" }];

it("getRadar gerçek API'den RadarPoint[] döndürür", async () => {
  vi.spyOn(global, "fetch").mockResolvedValue(new Response(JSON.stringify(sampleRadar)));
  const r = await httpApi.getRadar();
  expect(r[0].name).toBe("A");
});

it("LIVE_ENDPOINTS getKpis ve getRadar içerir", () => {
  expect(LIVE_ENDPOINTS.has("getKpis")).toBe(true);
  expect(LIVE_ENDPOINTS.has("getRadar")).toBe(true);
});
