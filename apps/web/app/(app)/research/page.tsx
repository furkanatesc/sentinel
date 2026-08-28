import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { ResearchContent } from "@/components/research/ResearchContent";

export default async function ResearchPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({
    queryKey: qk.researchSuggestions,
    queryFn: () => getApi().getResearchSuggestions(),
  });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <ResearchContent />
    </HydrationBoundary>
  );
}
