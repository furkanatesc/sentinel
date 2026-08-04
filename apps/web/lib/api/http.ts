import type { SentinelApi } from "./contract";
import type { StrategyRow } from "./types";

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

export const httpApi: SentinelApi = {
  getKpis: notReady,
  getTokens: notReady,
  getAlerts: notReady,
  getRadar: notReady,
  getToken: notReady,
  getEvents: notReady,
  getWalletGraph: notReady,
  getCreators: notReady,
  getCreator: notReady,
  getStrategies: () => getJson<StrategyRow[]>("/api/strategies"),
  getStrategy: notReady,
  getPortfolio: notReady,
  getPositions: notReady,
  getCandles: notReady,
  getMarketData: notReady,
  getOrders: notReady,
  getTransactions: notReady,
  getTradeLogs: notReady,
  runBacktest: notReady,
  subscribeTokens: () => () => {},
  subscribeAlerts: () => () => {},
  subscribeEvents: () => () => {},
};
