import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { PositionsContent } from "@/components/position/PositionsContent";

export default async function PositionsPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.positions, queryFn: () => getApi().getPositions() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <PositionsContent />
    </HydrationBoundary>
  );
}
