import { render, screen } from "@testing-library/react";
import { EventDetailDrawer } from "./EventDetailDrawer";
import type { FeedEvent } from "@/lib/api/types";

const ev: FeedEvent = { id: "x", type: "whale_buy", symbol: "PULSE", mint: "9x..4Fk2", launchpad: "Raydium", dex: "Orca", liquidity: 82400, creatorScore: 82, riskLevel: "good", tokenAgeSeconds: 30, volume5m: 41200, holderGrowthPct: 40, severity: "positive", detail: "detay", time: "az önce", ts: 1, watchlisted: false };

test("open drawer shows event + token link", async () => {
  render(<EventDetailDrawer event={ev} onClose={() => {}} />);
  // Sheet content is portaled and mounts after the open animation effect kicks in,
  // so assert with findBy* rather than the synchronous getBy* queries.
  expect(await screen.findByText("Balina Alımı")).toBeInTheDocument();
  const link = await screen.findByText("Token Detayına Git");
  expect(link.closest("a")!.getAttribute("href")).toBe("/tokens/PULSE");
});

test("closed drawer (event null) renders no token link", () => {
  render(<EventDetailDrawer event={null} onClose={() => {}} />);
  expect(screen.queryByText("Token Detayına Git")).not.toBeInTheDocument();
});
