import { render, screen } from "@testing-library/react";
import { StrategyPerformancePanel } from "./StrategyPerformancePanel";
import { BacktestSummaryPanel } from "./BacktestSummaryPanel";
import type { StrategyPerformance, BacktestSummary } from "@/lib/api/types";

const perf: StrategyPerformance = {
  winRatePct: 62, profitFactor: 2.1, maxDrawdownPct: 14, sharpe: 1.8, sortino: 2.3,
  totalTrades: 220, netPnlSol: 128.4, expectancy: 0.4,
};
const bt: BacktestSummary = {
  netPnlSol: 128.4, winRatePct: 62, profitFactor: 2.1, sharpe: 1.8, maxDrawdownPct: 14,
  trades: 220, avgHoldingHours: 6, rugExposurePct: 3,
};

test("performance panel shows sharpe and win rate tiles", () => {
  render(<StrategyPerformancePanel performance={perf} />);
  expect(screen.getByText("Sharpe")).toBeInTheDocument();
  expect(screen.getByText("1.8")).toBeInTheDocument();
});

test("backtest panel shows trades and rug exposure", () => {
  render(<BacktestSummaryPanel backtest={bt} />);
  expect(screen.getByText("İşlem Sayısı")).toBeInTheDocument();
  expect(screen.getByText("Rug Maruziyeti")).toBeInTheDocument();
});
