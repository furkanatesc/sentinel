import { filterEvents } from "./filter";
import { EMPTY_FILTERS } from "@/lib/api/types";
import type { FeedEvent } from "@/lib/api/types";

const ev = (o: Partial<FeedEvent>): FeedEvent => ({
  id: "1", type: "new_mint", symbol: "AAA", mint: "m", launchpad: "Pump.fun", dex: "Raydium",
  liquidity: 50000, creatorScore: 60, riskLevel: "medium", tokenAgeSeconds: 100,
  volume5m: 10000, holderGrowthPct: 20, severity: "info", detail: "d", time: "az önce", ts: 0, watchlisted: false, ...o,
});
const list = [
  ev({ id: "a", type: "whale_buy", riskLevel: "strong", launchpad: "Raydium", dex: "Orca", liquidity: 200000, creatorScore: 90, tokenAgeSeconds: 30, volume5m: 90000, holderGrowthPct: 60, watchlisted: true }),
  ev({ id: "b", type: "liquidity_removed", riskLevel: "critical", launchpad: "Pump.fun", dex: "Raydium", liquidity: 4000, creatorScore: 17, tokenAgeSeconds: 500, volume5m: 3000, holderGrowthPct: 2, watchlisted: false }),
];

test("EMPTY_FILTERS returns everything", () => {
  expect(filterEvents(list, EMPTY_FILTERS)).toHaveLength(2);
});
test("type filter", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, types: ["whale_buy"] }).map(e => e.id)).toEqual(["a"]);
});
test("risk filter", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, risks: ["critical"] }).map(e => e.id)).toEqual(["b"]);
});
test("launchpad + dex filter", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, launchpad: "Raydium" }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, dex: "Raydium" }).map(e => e.id)).toEqual(["b"]);
});
test("numeric thresholds", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, minLiquidity: 100000 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, minCreatorScore: 50 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, maxAgeSeconds: 60 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, minVolume: 50000 }).map(e => e.id)).toEqual(["a"]);
  expect(filterEvents(list, { ...EMPTY_FILTERS, minHolderGrowth: 30 }).map(e => e.id)).toEqual(["a"]);
});
test("watchlistOnly", () => {
  expect(filterEvents(list, { ...EMPTY_FILTERS, watchlistOnly: true }).map(e => e.id)).toEqual(["a"]);
});
