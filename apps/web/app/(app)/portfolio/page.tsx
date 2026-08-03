import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { PortfolioContent } from "@/components/portfolio/PortfolioContent";

export default async function PortfolioPage() {
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.portfolio, queryFn: () => getApi().getPortfolio() }),
    queryClient.prefetchQuery({ queryKey: qk.positions, queryFn: () => getApi().getPositions() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <PortfolioContent />
    </HydrationBoundary>
  );
}
