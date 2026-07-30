import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { OverviewContent } from "@/components/dashboard/OverviewContent";

export default async function OverviewPage() {
  const queryClient = getQueryClient();
  const api = getApi();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.kpis, queryFn: () => api.getKpis() }),
    queryClient.prefetchQuery({ queryKey: qk.tokens, queryFn: () => api.getTokens() }),
    queryClient.prefetchQuery({ queryKey: qk.alerts, queryFn: () => api.getAlerts() }),
    queryClient.prefetchQuery({ queryKey: qk.radar, queryFn: () => api.getRadar() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <OverviewContent />
    </HydrationBoundary>
  );
}
