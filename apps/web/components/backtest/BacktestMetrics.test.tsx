import { render, screen } from "@testing-library/react";
import { BacktestMetrics } from "./BacktestMetrics";
import type { BacktestMetrics as Metrics } from "@/lib/api/types";

const m: Metrics = {
  netPnlSol: 128.4, winRatePct: 62, profitFactor: 2.1, sharpe: 1.8, sortino: 2.3,
  maxDrawdownPct: 14, avgTradeSol: 0.58, rugExposurePct: 3, trades: 220, avgHoldingHours: 6,
};

test("renders all 10 metric labels", () => {
  render(<BacktestMetrics metrics={m} />);
  expect(screen.getByText("Net PnL (SOL)")).toBeInTheDocument();
  expect(screen.getByText("Sortino")).toBeInTheDocument();
  expect(screen.getByText("İşlem Sayısı")).toBeInTheDocument();
});
