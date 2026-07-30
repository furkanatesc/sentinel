import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { TokenDetailContent } from "@/components/token/TokenDetailContent";

export default async function TokenDetailPage({ params }: { params: Promise<{ mint: string }> }) {
  const { mint } = await params;
  const queryClient = getQueryClient();
  try {
    await queryClient.prefetchQuery({ queryKey: qk.token(mint), queryFn: () => getApi().getToken(mint) });
  } catch {
    // bilinmeyen token — client tarafında isError ile ele alınır
  }
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <TokenDetailContent mint={mint} />
    </HydrationBoundary>
  );
}
