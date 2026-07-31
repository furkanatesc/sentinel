import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useStrategies, useStrategy } from "./queries";

function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("useStrategies loads rows", async () => {
  const { result } = renderHook(() => useStrategies(), { wrapper });
  await waitFor(() => expect(result.current.data?.length).toBeGreaterThan(0));
});

test("useStrategy loads a detail", async () => {
  const { result } = renderHook(() => useStrategy("momentum-scalp"), { wrapper });
  await waitFor(() => expect(result.current.data?.id).toBe("momentum-scalp"));
});
