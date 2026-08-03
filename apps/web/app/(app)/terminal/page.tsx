import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { TerminalContent } from "@/components/terminal/TerminalContent";
import { DEFAULT_TERMINAL_MINT } from "@/lib/terminal/order-defs";

export default async function TerminalPage() {
  const queryClient = getQueryClient();
  const api = getApi();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.marketData(DEFAULT_TERMINAL_MINT), queryFn: () => api.getMarketData(DEFAULT_TERMINAL_MINT) }),
    queryClient.prefetchQuery({ queryKey: qk.candles(DEFAULT_TERMINAL_MINT), queryFn: () => api.getCandles(DEFAULT_TERMINAL_MINT) }),
    queryClient.prefetchQuery({ queryKey: qk.tokens, queryFn: () => api.getTokens() }),
    queryClient.prefetchQuery({ queryKey: qk.orders, queryFn: () => api.getOrders() }),
    queryClient.prefetchQuery({ queryKey: qk.transactions, queryFn: () => api.getTransactions() }),
    queryClient.prefetchQuery({ queryKey: qk.tradeLogs, queryFn: () => api.getTradeLogs() }),
    queryClient.prefetchQuery({ queryKey: qk.positions, queryFn: () => api.getPositions() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <TerminalContent />
    </HydrationBoundary>
  );
}
