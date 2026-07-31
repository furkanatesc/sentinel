import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { StrategyDetailContent } from "@/components/strategy/StrategyDetailContent";

export default async function StrategyDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const queryClient = getQueryClient();
  try {
    await queryClient.prefetchQuery({ queryKey: qk.strategy(id), queryFn: () => getApi().getStrategy(id) });
  } catch {
    // bilinmeyen strateji — client tarafında isError ile ele alınır
  }
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <StrategyDetailContent id={id} />
    </HydrationBoundary>
  );
}
