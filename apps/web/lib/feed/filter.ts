import type { FeedEvent, FeedFilters } from "@/lib/api/types";

export function filterEvents(events: FeedEvent[], f: FeedFilters): FeedEvent[] {
  return events.filter((e) =>
    (f.types.length === 0 || f.types.includes(e.type)) &&
    (f.risks.length === 0 || f.risks.includes(e.riskLevel)) &&
    (f.launchpad === "all" || e.launchpad === f.launchpad) &&
    (f.dex === "all" || e.dex === f.dex) &&
    e.liquidity >= f.minLiquidity &&
    e.creatorScore >= f.minCreatorScore &&
    (f.maxAgeSeconds === null || e.tokenAgeSeconds <= f.maxAgeSeconds) &&
    e.volume5m >= f.minVolume &&
    e.holderGrowthPct >= f.minHolderGrowth &&
    (!f.watchlistOnly || e.watchlisted)
  );
}
