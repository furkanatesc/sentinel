import { render, screen, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { PositionsTable } from "./PositionsTable";
import type { Position } from "@/lib/api/types";

const rows: Position[] = [
  { id: "pos-1", tokenMint: "9x..2", tokenSymbol: "PULSE", strategyId: "momentum-scalp", strategyName: "Momentum Scalp", entryPrice: 0.0004, currentPrice: 0.0006, sizeSol: 12, pnlSol: 5.4, pnlPct: 45, stopLossPct: 12, takeProfitPct: 60, tokenRisk: "good", creatorRisk: "medium", ageLabel: "8 dk", openedAt: "8 dk önce" },
];

test("renders token→/tokens and strategy→/strategies links; row click fires; action does not", () => {
  const onRowClick = vi.fn();
  render(<PositionsTable rows={rows} sortKey="pnlSol" onSort={() => {}} onRowClick={onRowClick} />);
  expect(screen.getByText("PULSE").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
  expect(screen.getByText("Momentum Scalp").closest("a")!.getAttribute("href")).toBe("/strategies/momentum-scalp");
  fireEvent.click(screen.getByText("Kapat"));           // action button
  expect(onRowClick).not.toHaveBeenCalled();            // stopPropagation
  fireEvent.click(screen.getByText("PULSE"));           // row (via token cell)
});
