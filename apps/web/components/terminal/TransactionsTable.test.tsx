import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TransactionsTable } from "./TransactionsTable";

test("lists transactions with an explorer link", async () => {
  render(<QueryClientProvider client={getQueryClient()}><TransactionsTable /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByRole("link").length).toBeGreaterThan(0));
  expect(screen.getAllByRole("link")[0]).toHaveAttribute("href", expect.stringContaining("solscan.io"));
});
