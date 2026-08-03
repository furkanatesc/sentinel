import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { PortfolioContent } from "./PortfolioContent";

test("composes KPIs, charts and open-positions summary", async () => {
  render(<QueryClientProvider client={getQueryClient()}><PortfolioContent /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("Portföy")).toBeInTheDocument());
  expect(screen.getByText("Toplam Değer")).toBeInTheDocument();
  expect(screen.getByText("Risk Dağılımı")).toBeInTheDocument();
  expect(screen.getByText("Açık Pozisyonlar")).toBeInTheDocument();
});
