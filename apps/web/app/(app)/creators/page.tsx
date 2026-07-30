import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { CreatorsList } from "@/components/creator/CreatorsList";

export default async function CreatorsPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.creators, queryFn: () => getApi().getCreators() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <CreatorsList />
    </HydrationBoundary>
  );
}
