import type { SentinelApi } from "./contract";
import type { StrategyRow, FeedEvent, TokenRow, TokenDetail, CreatorRow, CreatorProfile, Kpi, RadarPoint, WalletGraph, SystemHealth, BacktestParams, BacktestResult } from "./types";
import { wsSubscribe } from "./ws";

// TODO(backend): AWS REST + WebSocket implementasyonu. Endpoint aileleri
// ROADMAP servislerine maplenir (tokens→discovery, alerts→alert engine, ...).
const notReady = () => Promise.reject(new Error("httpApi not implemented — backend not connected yet"));

function apiBase(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) throw new Error("NEXT_PUBLIC_API_BASE_URL is not set");
  return base.replace(/\/$/, "");
}

async function getJson<T>(path: string): Promise<T> {
  const res = await fetch(`${apiBase()}${path}`, { headers: { accept: "application/json" } });
  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);
  return (await res.json()) as T;
}

async function postJson<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${apiBase()}${path}`, {
    method: "POST",
    headers: { accept: "application/json", "content-type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`API ${path} failed: ${res.status}`);
  return (await res.json()) as T;
}

export const httpApi: SentinelApi = {
  getKpis: () => getJson<Kpi[]>("/api/kpis"),
  getTokens: () => getJson<TokenRow[]>("/api/tokens"),
  getAlerts: notReady,
  // Alert kuralları + bildirim config'i backend Alt-proje 3'e bağlı; LIVE_ENDPOINTS'te
  // olmadıkları için hibrit adapter mock'a route eder — httpApi tarafı notReady kalır.
  getAlertRules: notReady,
  getNotificationConfig: notReady,
  getRadar: () => getJson<RadarPoint[]>("/api/radar"),
  getToken: (mint: string) => getJson<TokenDetail>(`/api/token/${encodeURIComponent(mint)}`),
  getEvents: () => getJson<FeedEvent[]>("/api/events"),
  getWalletGraph: () => getJson<WalletGraph>("/api/wallet-graph"),
  getAuthorityGraph: () => getJson<WalletGraph>("/api/authority-graph"),
  getCreators: () => getJson<CreatorRow[]>("/api/creators"),
  getCreator: (address: string) => getJson<CreatorProfile>(`/api/creator/${encodeURIComponent(address)}`),
  getStrategies: () => getJson<StrategyRow[]>("/api/strategies"),
  getStrategy: notReady,
  getPortfolio: notReady,
  getPositions: notReady,
  getCandles: notReady,
  getMarketData: notReady,
  getOrders: notReady,
  getTransactions: notReady,
  getTradeLogs: notReady,
  runBacktest: (params: BacktestParams) => postJson<BacktestResult>("/api/backtest", params),
  getSystemHealth: () => getJson<SystemHealth>("/api/system-health"),
  getResearchSuggestions: notReady,
  streamResearchAnswer: () => { throw new Error("httpApi not implemented — backend not connected yet"); },
  // subscribeTokens SentinelApi'de tam TokenRow[] snapshot'ı ile çağrılır (bkz contract.ts);
  // subscribeEvents ise tekil FeedEvent ile. WS "tokens" topic payload'ı da bu yüzden dizi.
  subscribeTokens: (cb) => wsSubscribe<TokenRow[]>("tokens", cb),
  subscribeAlerts: () => () => {},
  subscribeEvents: (cb) => wsSubscribe<FeedEvent>("events", cb),
};
