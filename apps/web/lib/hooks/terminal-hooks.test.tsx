import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useMarketData, useOrders } from "./queries";

function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("useMarketData resolves market data", async () => {
  const { result } = renderHook(() => useMarketData("PULSE"), { wrapper });
  await waitFor(() => expect(result.current.data?.symbol).toBe("PULSE"));
});

test("useOrders resolves a non-empty list", async () => {
  const { result } = renderHook(() => useOrders(), { wrapper });
  await waitFor(() => expect((result.current.data ?? []).length).toBeGreaterThan(0));
});
