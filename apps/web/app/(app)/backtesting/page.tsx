import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { BacktestContent } from "@/components/backtest/BacktestContent";

export default async function BacktestingPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.strategies, queryFn: () => getApi().getStrategies() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <BacktestContent />
    </HydrationBoundary>
  );
}
