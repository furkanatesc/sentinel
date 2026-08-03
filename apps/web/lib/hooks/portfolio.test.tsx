import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { usePortfolio, usePositions } from "./queries";

function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("usePortfolio loads the overview", async () => {
  const { result } = renderHook(() => usePortfolio(), { wrapper });
  await waitFor(() => expect(result.current.data?.summary).toBeDefined());
});

test("usePositions loads rows", async () => {
  const { result } = renderHook(() => usePositions(), { wrapper });
  await waitFor(() => expect(result.current.data?.length).toBeGreaterThan(0));
});
