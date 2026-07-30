import type { Kpi, TokenRow, AlertEvent, RadarPoint } from "./types";

export interface SentinelApi {
  getKpis(): Promise<Kpi[]>;
  getTokens(): Promise<TokenRow[]>;
  getAlerts(): Promise<AlertEvent[]>;
  getRadar(): Promise<RadarPoint[]>;
  /** Real-time seam — mock: interval, http: WebSocket. Returns unsubscribe fn. */
  subscribeTokens(cb: (tokens: TokenRow[]) => void): () => void;
  subscribeAlerts(cb: (alert: AlertEvent) => void): () => void;
}
