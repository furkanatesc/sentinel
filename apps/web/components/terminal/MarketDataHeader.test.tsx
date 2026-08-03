import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { MarketDataHeader } from "./MarketDataHeader";

test("shows symbol, price and score labels", async () => {
  render(
    <QueryClientProvider client={getQueryClient()}>
      <MarketDataHeader mint="PULSE" />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("PULSE")).toBeInTheDocument());
  expect(screen.getByText("Likidite")).toBeInTheDocument();
  expect(screen.getByText(/Token \d+/)).toBeInTheDocument();
});
