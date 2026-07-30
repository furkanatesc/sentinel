import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { WalletGraphContent } from "@/components/graph/WalletGraphContent";

export default async function WalletGraphPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({ queryKey: qk.walletGraph, queryFn: () => getApi().getWalletGraph() });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <WalletGraphContent />
    </HydrationBoundary>
  );
}
