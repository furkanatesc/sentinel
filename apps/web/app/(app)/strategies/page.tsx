import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { StrategiesListContent } from "@/components/strategy/StrategiesListContent";

export default async function StrategiesPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.strategies, queryFn: () => getApi().getStrategies() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <StrategiesListContent />
    </HydrationBoundary>
  );
}
