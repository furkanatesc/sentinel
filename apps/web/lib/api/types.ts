import type { RiskLevel, AlertSeverity } from "@/lib/format";
import type { ScoreKey } from "@/lib/token/score-defs";
import type { RiskSeverity } from "@/lib/format";

export interface Kpi {
  id: string;
  label: string;
  value: string;
  change: number;
  spark: number[];
  updated: string;
  tone?: "positive" | "critical" | "warning" | "neutral";
}

export interface TokenRow {
  id: string;
  name: string;
  symbol: string;
  mint: string;
  ageSeconds: number;
  price: number;
  liquidity: number;
  vol5m: number;
  holders: number;
  creatorScore: number;
  safetyScore: number;
  momentum: number;
  spark: number[];
  signal: "buy" | "watch" | "avoid" | null;
  watchlisted: boolean;
}

export interface AlertEvent {
  id: string;
  type: string;
  token: string;
  detail: string;
  severity: import("@/lib/format").AlertSeverity;
  time: string;
}

export interface RadarPoint {
  x: number;
  y: number;
  z: number;
  name: string;
  level: RiskLevel;
}

export interface ScoreBreakdownItem { label: string; weight: number; detail: string; }
export interface ScoreDetail { key: ScoreKey; value: number; confidence: number; updatedAt: string; breakdown: ScoreBreakdownItem[]; }
export interface RiskItem { id: string; title: string; severity: RiskSeverity; description: string; evidence?: string; firstSeen: string; lastSeen: string; }
export interface RiskGroups { contract: RiskItem[]; market: RiskItem[]; creator: RiskItem[]; }
export interface SeriesPoint { t: number; v: number; }
export interface TokenMetrics {
  holders: number; uniqueBuyers: number; buyRatio: number; sellRatio: number;
  creatorHoldingPct: number; top10HolderPct: number; sniperPct: number; botActivityPct: number;
}
export interface TokenDetail {
  id: string; name: string; symbol: string; mint: string;
  ageSeconds: number; price: number; priceChange24h: number;
  marketCap: number; liquidity: number; volume24h: number;
  scores: Record<ScoreKey, ScoreDetail>;
  metrics: TokenMetrics;
  series: { price: SeriesPoint[]; liquidity: SeriesPoint[]; volume: SeriesPoint[]; holders: SeriesPoint[] };
  risks: RiskGroups;
}

export type CreatorOutcome = "active" | "graduated" | "dumped" | "rug" | "dead";
export type LiquidityStatus = "locked" | "unlocked" | "removed";

export type EventType =
  | "new_mint" | "metadata_created" | "pool_created" | "first_swap"
  | "liquidity_added" | "liquidity_removed" | "creator_sell" | "whale_buy"
  | "suspicious_cluster" | "score_change" | "strategy_signal";

export interface FeedEvent {
  id: string; type: EventType;
  symbol: string; mint: string;
  launchpad: string; dex: string;
  liquidity: number; creatorScore: number;
  riskLevel: RiskLevel; tokenAgeSeconds: number;
  volume5m: number; holderGrowthPct: number;
  severity: AlertSeverity; detail: string; time: string; ts: number;
  watchlisted: boolean;
}

export interface FeedFilters {
  types: EventType[]; risks: RiskLevel[];
  launchpad: string; dex: string;
  minLiquidity: number; minCreatorScore: number;
  maxAgeSeconds: number | null; minVolume: number; minHolderGrowth: number;
  watchlistOnly: boolean;
}

export const EMPTY_FILTERS: FeedFilters = {
  types: [], risks: [], launchpad: "all", dex: "all",
  minLiquidity: 0, minCreatorScore: 0, maxAgeSeconds: null,
  minVolume: 0, minHolderGrowth: 0, watchlistOnly: false,
};

export interface CreatorRow {
  address: string; label?: string; reputationScore: number; riskLevel: RiskLevel;
  totalTokens: number; activeTokens: number; ruggedTokens: number; successRatePct: number; realizedPnlSol: number;
}
export interface CreatorTokenHistoryItem {
  id: string; symbol: string; mint: string; createdAt: string;
  peakMarketCap: number; currentMarketCap: number; maxDrawdownPct: number;
  liquidityStatus: LiquidityStatus; creatorSellPct: number; outcome: CreatorOutcome; riskFlags: string[];
}
export interface CreatorBehavior {
  deployFrequency: string; avgFirstSellMinutes: number; repeatedFunders: string[];
  similarMetadata: boolean; sameSocial: boolean; sameLiquidityPattern: boolean;
}
export interface CreatorMetrics {
  totalTokens: number; activeTokens: number; ruggedTokens: number; avgLifetimeHours: number;
  avgPeakMarketCap: number; realizedPnlSol: number; successRatePct: number; avgFirstSellMinutes: number;
}
export interface CreatorProfile {
  address: string; label?: string; walletAgeDays: number; firstSeen: string;
  reputation: ScoreDetail; riskLevel: RiskLevel; metrics: CreatorMetrics;
  history: CreatorTokenHistoryItem[]; behavior: CreatorBehavior;
}

export type GraphNodeType =
  | "creator_wallet" | "funding_wallet" | "authority_wallet" | "token" | "liquidity_pool"
  | "trader_wallet" | "smart_wallet" | "suspicious_wallet" | "exchange_wallet";
export type GraphEdgeType =
  | "funded" | "created" | "bought" | "sold" | "transferred"
  | "provided_liquidity" | "removed_liquidity" | "shares_funder" | "controls_authority";

export interface GraphNode {
  id: string; type: GraphNodeType; label: string;
  address?: string; riskLevel: RiskLevel; balanceSol?: number; firstSeen: string; lastSeen: string;
}
export interface GraphEdge { id: string; source: string; target: string; type: GraphEdgeType; role?: "mint" | "freeze" | "both"; }
export interface WalletGraph { nodes: GraphNode[]; edges: GraphEdge[]; }
export interface GraphFilters { relationships: GraphEdgeType[]; risks: RiskLevel[]; }
export const EMPTY_GRAPH_FILTERS: GraphFilters = { relationships: [], risks: [] };

// --- Strategies (Increment 6) ---
export type StrategyStatus = "draft" | "backtesting" | "paper" | "shadow" | "live" | "paused" | "archived";
export type ConditionOp = ">" | "<" | ">=" | "<=" | "==";

export interface StrategyCondition { metric: string; op: ConditionOp; value: number; unit?: string; }

export interface StrategyRow {
  id: string; name: string; status: StrategyStatus; timeframe: string;
  winRatePct: number; profitFactor: number; maxDrawdownPct: number; totalTrades: number;
  netPnlSol: number; lastSignal: string;
}

export interface StrategyPerformance {
  winRatePct: number; profitFactor: number; maxDrawdownPct: number; sharpe: number; sortino: number;
  totalTrades: number; netPnlSol: number; expectancy: number;
}

export interface EquityPoint { t: number; v: number; }

export interface BacktestSummary {
  netPnlSol: number; winRatePct: number; profitFactor: number; sharpe: number; maxDrawdownPct: number;
  trades: number; avgHoldingHours: number; rugExposurePct: number;
}

export interface StrategyVersion { version: string; date: string; note: string; }
export interface AuditEntry { time: string; action: string; detail: string; }
export interface StrategyRisk { riskPerTradePct: number; stopLossPct: number; takeProfitLevels: number[]; maxDrawdownStopPct: number; }
export interface StrategySizing { model: string; sizePct: number; }

export interface StrategyDetail {
  id: string; name: string; status: StrategyStatus; timeframe: string; description: string;
  entry: StrategyCondition[]; exit: StrategyCondition[];
  risk: StrategyRisk; sizing: StrategySizing;
  supportedLaunchpads: string[]; minScores: { creator: number; safety: number };
  performance: StrategyPerformance; equityCurve: EquityPoint[]; backtest: BacktestSummary;
  versions: StrategyVersion[]; audit: AuditEntry[];
}

// --- Portfolio & Positions (Increment 7) ---
export interface PortfolioSummary {
  totalValueSol: number; availableSol: number; investedSol: number;
  realizedPnlSol: number; unrealizedPnlSol: number; dailyPnlSol: number;
  maxDrawdownPct: number; riskExposurePct: number; rugExposurePct: number;
}
export interface StrategyPnl { strategyId: string; name: string; pnlSol: number; }
export interface AllocationSlice { label: string; pct: number; color: string; }
export interface WinLossBucket { label: string; count: number; }
export interface PortfolioOverview {
  summary: PortfolioSummary;
  equityCurve: EquityPoint[];
  pnlByStrategy: StrategyPnl[];
  riskAllocation: AllocationSlice[];
  winLoss: WinLossBucket[];
}
export interface Position {
  id: string; tokenMint: string; tokenSymbol: string;
  strategyId: string; strategyName: string;
  entryPrice: number; currentPrice: number; sizeSol: number;
  pnlSol: number; pnlPct: number;
  stopLossPct: number; takeProfitPct: number;
  tokenRisk: RiskLevel; creatorRisk: RiskLevel;
  ageLabel: string; openedAt: string;
}

// --- Trading Terminal (Increment 8) ---
export interface Candle { time: number; open: number; high: number; low: number; close: number; }
export interface MarketData {
  mint: string; symbol: string;
  price: number; change24hPct: number;
  liquiditySol: number; volume24hSol: number; marketCapSol: number;
  tokenScore: number; creatorScore: number;
}
export type OrderSide = "buy" | "sell";
export type OrderType = "market" | "limit";
export type OrderStatus = "open" | "filled" | "cancelled";
export interface Order {
  id: string; tokenSymbol: string; tokenMint: string;
  side: OrderSide; type: OrderType; status: OrderStatus;
  price: number; amountSol: number; createdAt: string;
}
export interface Txn {
  id: string; hash: string; kind: "buy" | "sell" | "approve";
  tokenSymbol: string; amountSol: number; status: "success" | "pending" | "failed"; time: string;
}
export interface TradeLog { id: string; level: "info" | "warn" | "error"; message: string; time: string; }

// --- Backtesting (Increment 9) ---
export interface BacktestParams {
  strategyId: string; rangePreset: string;
  initialCapitalSol: number; maxPositions: number;
  slippageModel: string; priorityFee: number;
  latencyModel: string; liquidityModel: string;
  minCreatorScore: number; minTokenSafety: number;
}
export interface BacktestMetrics {
  netPnlSol: number; winRatePct: number; profitFactor: number; sharpe: number; sortino: number;
  maxDrawdownPct: number; avgTradeSol: number; rugExposurePct: number; trades: number; avgHoldingHours: number;
}
export interface MonthlyReturn { label: string; pct: number; }
export interface DistributionBucket { label: string; count: number; }
export interface ScorePnl { scoreBucket: string; pnlSol: number; }
export interface DrawdownPoint { t: number; v: number; }
export interface BacktestTrade { time: number; price: number; side: "buy" | "sell"; pnlSol: number; }
export interface BacktestResult {
  metrics: BacktestMetrics;
  equityCurve: EquityPoint[];
  drawdown: DrawdownPoint[];
  monthlyReturns: MonthlyReturn[];
  tradeDistribution: DistributionBucket[];
  pnlByScore: ScorePnl[];
  priceSeries: EquityPoint[];
  trades: BacktestTrade[];
}

// --- System Health ---
export type WorkerState = "off" | "starting" | "ok" | "degraded" | "stalled";

export interface WorkerStatus {
  name: string;
  state: WorkerState;
  lastRunAt: string; // RFC3339 or ""
  lastErr: string;
  cyclesRun: number;
  itemsProcessed: number;
  intervalSec: number;
}

export interface SystemHealth {
  uptimeSec: number;
  version: string;
  dbOk: boolean;
  dbLatencyMs: number;
  wsClients: number;
  workers: WorkerStatus[];
  gates: Record<string, boolean>;
}
