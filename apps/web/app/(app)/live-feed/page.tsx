import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { LiveFeedContent } from "@/components/feed/LiveFeedContent";

export default async function LiveFeedPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.events, queryFn: () => getApi().getEvents() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <LiveFeedContent />
    </HydrationBoundary>
  );
}
