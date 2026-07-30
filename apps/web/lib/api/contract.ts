import type { Kpi, TokenRow, AlertEvent, RadarPoint, TokenDetail, FeedEvent } from "./types";

export interface SentinelApi {
  getKpis(): Promise<Kpi[]>;
  getTokens(): Promise<TokenRow[]>;
  getAlerts(): Promise<AlertEvent[]>;
  getRadar(): Promise<RadarPoint[]>;
  getToken(idOrMint: string): Promise<TokenDetail>;
  getEvents(): Promise<FeedEvent[]>;
  /** Real-time seam — mock: interval, http: WebSocket. Returns unsubscribe fn. */
  subscribeTokens(cb: (tokens: TokenRow[]) => void): () => void;
  subscribeAlerts(cb: (alert: AlertEvent) => void): () => void;
  subscribeEvents(cb: (e: FeedEvent) => void): () => void;
}
