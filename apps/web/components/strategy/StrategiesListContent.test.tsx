import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { StrategiesListContent } from "./StrategiesListContent";

function renderList() {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <StrategiesListContent />
    </QueryClientProvider>
  );
}

test("renders strategy cards from the seam", async () => {
  renderList();
  await waitFor(() => expect(screen.getByText("Momentum Scalp")).toBeInTheDocument());
});

test("status filter narrows the visible cards", async () => {
  renderList();
  await waitFor(() => expect(screen.getByText("Momentum Scalp")).toBeInTheDocument());
  // "Canlı" filter keeps the live strategy, drops an archived one
  fireEvent.click(screen.getByRole("button", { name: "Canlı" }));
  await waitFor(() => {
    expect(screen.getByText("Momentum Scalp")).toBeInTheDocument();
    expect(screen.queryByText("Eski Sniper v1")).not.toBeInTheDocument();
  });
});
