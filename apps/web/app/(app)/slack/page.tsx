import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import SlackContent from "./SlackContent";

export default async function SlackPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({
    queryKey: qk.notificationConfig,
    queryFn: () => getApi().getNotificationConfig(),
  });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <SlackContent />
    </HydrationBoundary>
  );
}
