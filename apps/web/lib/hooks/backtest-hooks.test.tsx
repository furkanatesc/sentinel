import { renderHook, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useBacktest } from "./queries";
import type { BacktestParams } from "@/lib/api/types";

const params: BacktestParams = {
  strategyId: "momentum-scalp", rangePreset: "30g", initialCapitalSol: 100, maxPositions: 5,
  slippageModel: "dynamic", priorityFee: 0.0001, latencyModel: "realistic", liquidityModel: "constrained",
  minCreatorScore: 60, minTokenSafety: 55,
};
function wrapper({ children }: { children: React.ReactNode }) {
  return <QueryClientProvider client={getQueryClient()}>{children}</QueryClientProvider>;
}

test("useBacktest does not fetch when params is null", () => {
  const { result } = renderHook(() => useBacktest(null), { wrapper });
  expect(result.current.fetchStatus).toBe("idle");
  expect(result.current.data).toBeUndefined();
});

test("useBacktest resolves a result when params are provided", async () => {
  const { result } = renderHook(() => useBacktest(params), { wrapper });
  await waitFor(() => expect(result.current.data?.metrics.trades).toBeGreaterThan(0));
});
