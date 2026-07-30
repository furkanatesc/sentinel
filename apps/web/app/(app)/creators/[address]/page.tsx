import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { CreatorProfileContent } from "@/components/creator/CreatorProfileContent";

export default async function CreatorProfilePage({ params }: { params: Promise<{ address: string }> }) {
  const { address } = await params;
  const queryClient = getQueryClient();
  try {
    await queryClient.prefetchQuery({ queryKey: qk.creator(address), queryFn: () => getApi().getCreator(address) });
  } catch {
    // bilinmeyen üretici — client tarafında isError ile ele alınır
  }
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <CreatorProfileContent address={address} />
    </HydrationBoundary>
  );
}
