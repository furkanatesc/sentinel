import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TradeLogsList } from "./TradeLogsList";

test("renders log messages", async () => {
  render(<QueryClientProvider client={getQueryClient()}><TradeLogsList /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByText(/Emir simülasyonu tamamlandı|Sinyal alındı/)).not.toHaveLength(0));
});
