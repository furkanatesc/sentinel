import { render, screen } from "@testing-library/react";
import { StrategyCard } from "./StrategyCard";
import type { StrategyRow } from "@/lib/api/types";

const row: StrategyRow = {
  id: "momentum-scalp", name: "Momentum Scalp", status: "live", timeframe: "1-5 dk",
  winRatePct: 62, profitFactor: 2.1, maxDrawdownPct: 14, totalTrades: 220,
  netPnlSol: 128.4, lastSignal: "3 dk önce",
};

test("shows name, status badge, metrics and links to the detail page", () => {
  render(<StrategyCard row={row} />);
  expect(screen.getByText("Momentum Scalp")).toBeInTheDocument();
  expect(screen.getByText("Canlı")).toBeInTheDocument();
  expect(screen.getByText(/62/)).toBeInTheDocument();
  const link = screen.getByRole("link");
  expect(link.getAttribute("href")).toBe("/strategies/momentum-scalp");
});
