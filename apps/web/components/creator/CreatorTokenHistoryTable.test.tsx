import { render, screen } from "@testing-library/react";
import { CreatorTokenHistoryTable } from "./CreatorTokenHistoryTable";
import type { CreatorTokenHistoryItem } from "@/lib/api/types";

const h: CreatorTokenHistoryItem = { id: "1", symbol: "PULSE", mint: "m", createdAt: "1g önce", peakMarketCap: 100000, currentMarketCap: 10000, maxDrawdownPct: 90, liquidityStatus: "removed", creatorSellPct: 40, outcome: "rug", riskFlags: [] };

test("renders row with outcome badge and token link", () => {
  render(<CreatorTokenHistoryTable history={[h]} />);
  expect(screen.getByText("Rug")).toBeInTheDocument();
  expect(screen.getByText("PULSE").closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
});
