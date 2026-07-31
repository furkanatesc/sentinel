import type { SentinelApi } from "./contract";

// TODO(backend): AWS REST + WebSocket implementasyonu. Endpoint aileleri
// ROADMAP servislerine maplenir (tokens→discovery, alerts→alert engine, ...).
const notReady = () => Promise.reject(new Error("httpApi not implemented — backend not connected yet"));

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
  getStrategies: notReady,
  getStrategy: notReady,
  subscribeTokens: () => () => {},
  subscribeAlerts: () => () => {},
  subscribeEvents: () => () => {},
};
