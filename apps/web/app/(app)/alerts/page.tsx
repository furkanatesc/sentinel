import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import AlertsContent from "./AlertsContent";

export default async function AlertsPage() {
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.alertRules, queryFn: () => getApi().getAlertRules() }),
    queryClient.prefetchQuery({ queryKey: qk.alerts, queryFn: () => getApi().getAlerts() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AlertsContent />
    </HydrationBoundary>
  );
}
