import { render, screen } from "@testing-library/react";
import { PositionDetailDrawer } from "./PositionDetailDrawer";
import type { Position } from "@/lib/api/types";

const p: Position = { id: "pos-1", tokenMint: "9x..2", tokenSymbol: "PULSE", strategyId: "momentum-scalp", strategyName: "Momentum Scalp", entryPrice: 0.0004, currentPrice: 0.0006, sizeSol: 12, pnlSol: 5.4, pnlPct: 45, stopLossPct: 12, takeProfitPct: 60, tokenRisk: "good", creatorRisk: "medium", ageLabel: "8 dk", openedAt: "8 dk önce" };

test("renders position breakdown when open", () => {
  render(<PositionDetailDrawer position={p} onClose={() => {}} />);
  expect(screen.getByText("PULSE")).toBeInTheDocument();
  expect(screen.getByText("Giriş / Güncel")).toBeInTheDocument();
  expect(screen.getByText("Stop-Loss / Take-Profit")).toBeInTheDocument();
});
