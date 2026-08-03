import { render, screen } from "@testing-library/react";
import { PortfolioKpis } from "./PortfolioKpis";
import type { PortfolioSummary } from "@/lib/api/types";

const summary: PortfolioSummary = {
  totalValueSol: 934.5, availableSol: 210, investedSol: 640,
  realizedPnlSol: 312.7, unrealizedPnlSol: 84.5, dailyPnlSol: -18.2,
  maxDrawdownPct: 22, riskExposurePct: 68, rugExposurePct: 4,
};

test("shows key portfolio KPI labels and values", () => {
  render(<PortfolioKpis summary={summary} />);
  expect(screen.getByText("Toplam Değer")).toBeInTheDocument();
  expect(screen.getByText("Günlük PnL")).toBeInTheDocument();
  expect(screen.getByText(/-18.2 SOL/)).toBeInTheDocument();
});
