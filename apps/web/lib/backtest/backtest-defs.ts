import type { BacktestParams, BacktestMetrics } from "@/lib/api/types";

export const RANGE_PRESETS: { key: string; label: string }[] = [
  { key: "7g", label: "Son 7 gün" }, { key: "30g", label: "Son 30 gün" },
  { key: "90g", label: "Son 90 gün" }, { key: "1y", label: "Son 1 yıl" },
];
export const SLIPPAGE_MODELS: { key: string; label: string }[] = [
  { key: "fixed", label: "Sabit" }, { key: "dynamic", label: "Dinamik" }, { key: "pessimistic", label: "Kötümser" },
];
export const LATENCY_MODELS: { key: string; label: string }[] = [
  { key: "low", label: "Düşük" }, { key: "realistic", label: "Gerçekçi" }, { key: "high", label: "Yüksek" },
];
export const LIQUIDITY_MODELS: { key: string; label: string }[] = [
  { key: "unconstrained", label: "Kısıtsız" }, { key: "constrained", label: "Likidite-oranlı" },
];

export const DEFAULT_BACKTEST_PARAMS: BacktestParams = {
  strategyId: "momentum-scalp", rangePreset: "30g", initialCapitalSol: 100, maxPositions: 5,
  slippageModel: "dynamic", priorityFee: 0.0001, latencyModel: "realistic", liquidityModel: "constrained",
  minCreatorScore: 60, minTokenSafety: 55,
};

export const BACKTEST_METRIC_DEFS: { key: keyof BacktestMetrics; label: string; kind: "pnl" | "pct" | "num" }[] = [
  { key: "netPnlSol", label: "Net PnL (SOL)", kind: "pnl" },
  { key: "winRatePct", label: "Kazanma Oranı", kind: "pct" },
  { key: "profitFactor", label: "Profit Factor", kind: "num" },
  { key: "sharpe", label: "Sharpe", kind: "num" },
  { key: "sortino", label: "Sortino", kind: "num" },
  { key: "maxDrawdownPct", label: "Maks. Drawdown", kind: "pct" },
  { key: "avgTradeSol", label: "Ort. İşlem (SOL)", kind: "pnl" },
  { key: "rugExposurePct", label: "Rug Maruziyeti", kind: "pct" },
  { key: "trades", label: "İşlem Sayısı", kind: "num" },
  { key: "avgHoldingHours", label: "Ort. Tutma (saat)", kind: "num" },
];
