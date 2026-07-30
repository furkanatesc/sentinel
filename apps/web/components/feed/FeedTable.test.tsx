import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FeedTable } from "./FeedTable";
import type { FeedEvent } from "@/lib/api/types";

const ev: FeedEvent = { id: "x", type: "whale_buy", symbol: "PULSE", mint: "9x..4Fk2", launchpad: "Raydium", dex: "Orca", liquidity: 82400, creatorScore: 82, riskLevel: "good", tokenAgeSeconds: 30, volume5m: 41200, holderGrowthPct: 40, severity: "positive", detail: "d", time: "az önce", ts: 1, watchlisted: false };

test("row click calls onRowClick with the event", async () => {
  const onRowClick = vi.fn();
  render(<FeedTable events={[ev]} onRowClick={onRowClick} />);
  await userEvent.click(screen.getByText("PULSE"));
  expect(onRowClick).toHaveBeenCalledWith(ev);
});

test("empty state when no events", () => {
  render(<FeedTable events={[]} onRowClick={() => {}} />);
  expect(screen.getByText("Filtrelere uygun event yok")).toBeInTheDocument();
});
