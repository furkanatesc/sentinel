import { render, screen, waitFor, fireEvent, within } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { PositionsContent } from "./PositionsContent";

function renderList() {
  return render(<QueryClientProvider client={getQueryClient()}><PositionsContent /></QueryClientProvider>);
}

test("renders positions from the seam", async () => {
  renderList();
  // The heading now renders only once data resolves (loading state shows a
  // Skeleton instead), mirroring PortfolioContent — so wait for it too.
  await waitFor(() => expect(screen.getByText("Pozisyonlar")).toBeInTheDocument());
  await waitFor(() => expect(screen.getAllByRole("row").length).toBeGreaterThan(1));
});

test("clicking a row opens the detail drawer", async () => {
  renderList();
  await waitFor(() => expect(screen.getAllByText("PULSE").length).toBeGreaterThan(0));
  const row = screen.getAllByText("PULSE")[0].closest("tr")!;
  // "PULSE" itself sits inside a <Link> that calls stopPropagation, so click a
  // non-link cell in the same row instead — the size cell text (e.g. "12 SOL").
  fireEvent.click(within(row).getByText(/SOL/));
  await waitFor(() => expect(screen.getByText("Giriş / Güncel")).toBeInTheDocument());
});
