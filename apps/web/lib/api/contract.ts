import type { Kpi, TokenRow, AlertEvent, AlertRule, NotificationConfig, RadarPoint, TokenDetail, FeedEvent, WalletGraph, CreatorRow, CreatorProfile, StrategyRow, StrategyDetail, PortfolioOverview, Position, Candle, MarketData, Order, Txn, TradeLog, BacktestParams, BacktestResult, SystemHealth, ResearchSuggestion, ResearchAnswer, ResearchSource } from "./types";
import type { AlertRuleDraft } from "@/lib/alerts/alert-defs";

export interface SentinelApi {
  getKpis(): Promise<Kpi[]>;
  getTokens(): Promise<TokenRow[]>;
  getAlerts(): Promise<AlertEvent[]>;
  getAlertRules(): Promise<AlertRule[]>;
  getNotificationConfig(): Promise<NotificationConfig>;
  // İlk mutation seam'i (kural CRUD — trade değil). Backend Alt-proje 3 persist.
  createAlertRule(draft: AlertRuleDraft): Promise<AlertRule>;
  setAlertRuleEnabled(id: string, enabled: boolean): Promise<void>;
  getRadar(): Promise<RadarPoint[]>;
  getToken(idOrMint: string): Promise<TokenDetail>;
  getEvents(): Promise<FeedEvent[]>;
  getWalletGraph(): Promise<WalletGraph>;
  getAuthorityGraph(): Promise<WalletGraph>;
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
  runBacktest(params: BacktestParams): Promise<BacktestResult>;
  getSystemHealth(): Promise<SystemHealth>;
  getResearchSuggestions(): Promise<ResearchSuggestion[]>;
  /** Streaming seam — subscribe pattern. onChunk = incremental text, onDone = final answer + sources. Returns cancel fn. */
  streamResearchAnswer(
    question: string,
    onChunk: (chunk: string) => void,
    onDone: (answer: ResearchAnswer) => void,
  ): () => void;
  /** Real-time seam — mock: interval, http: WebSocket. Returns unsubscribe fn. */
  subscribeTokens(cb: (tokens: TokenRow[]) => void): () => void;
  subscribeAlerts(cb: (alert: AlertEvent) => void): () => void;
  subscribeEvents(cb: (e: FeedEvent) => void): () => void;
}
