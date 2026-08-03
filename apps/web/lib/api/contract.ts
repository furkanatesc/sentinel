import type { Kpi, TokenRow, AlertEvent, RadarPoint, TokenDetail, FeedEvent, WalletGraph, CreatorRow, CreatorProfile, StrategyRow, StrategyDetail, PortfolioOverview, Position, Candle, MarketData, Order, Txn, TradeLog } from "./types";

export interface SentinelApi {
  getKpis(): Promise<Kpi[]>;
  getTokens(): Promise<TokenRow[]>;
  getAlerts(): Promise<AlertEvent[]>;
  getRadar(): Promise<RadarPoint[]>;
  getToken(idOrMint: string): Promise<TokenDetail>;
  getEvents(): Promise<FeedEvent[]>;
  getWalletGraph(): Promise<WalletGraph>;
  getCreators(): Promise<CreatorRow[]>;
  getCreator(address: string): Promise<CreatorProfile>;
  getStrategies(): Promise<StrategyRow[]>;
  getStrategy(id: string): Promise<StrategyDetail>;
  getPortfolio(): Promise<PortfolioOverview>;
  getPositions(): Promise<Position[]>;
  getCandles(mint: string): Promise<Candle[]>;
  getMarketData(mint: string): Promise<MarketData>;
  getOrders(): Promise<Order[]>;
  getTransactions(): Promise<Txn[]>;
  getTradeLogs(): Promise<TradeLog[]>;
  /** Real-time seam — mock: interval, http: WebSocket. Returns unsubscribe fn. */
  subscribeTokens(cb: (tokens: TokenRow[]) => void): () => void;
  subscribeAlerts(cb: (alert: AlertEvent) => void): () => void;
  subscribeEvents(cb: (e: FeedEvent) => void): () => void;
}
