import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { StrategyDetailContent } from "./StrategyDetailContent";

test("renders the composed detail (header + conditions + backtest section)", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <StrategyDetailContent id="momentum-scalp" />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("Momentum Scalp")).toBeInTheDocument());
  expect(screen.getByText("Giriş (IF)")).toBeInTheDocument();
  expect(screen.getByText("Çıkış (THEN)")).toBeInTheDocument();
  expect(screen.getByText("Backtest Özeti")).toBeInTheDocument();
});
