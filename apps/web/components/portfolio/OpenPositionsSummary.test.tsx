import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { OpenPositionsSummary } from "./OpenPositionsSummary";

test("lists open positions with a link to /positions", async () => {
  render(<QueryClientProvider client={getQueryClient()}><OpenPositionsSummary /></QueryClientProvider>);
  await waitFor(() => expect(screen.getByText("Açık Pozisyonlar")).toBeInTheDocument());
  const link = screen.getByText(/Tümü/).closest("a")!;
  expect(link.getAttribute("href")).toBe("/positions");
});
