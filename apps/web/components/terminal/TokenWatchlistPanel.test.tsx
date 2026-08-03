import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TokenWatchlistPanel } from "./TokenWatchlistPanel";

function renderPanel(onSelect = () => {}) {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <TokenWatchlistPanel activeMint="9xQeWv...4Fk2" onSelect={onSelect} />
    </QueryClientProvider>
  );
}

test("lists token symbols and reports selection", async () => {
  const onSelect = vi.fn();
  renderPanel(onSelect);
  await waitFor(() => expect(screen.getByText("NOVA")).toBeInTheDocument());
  fireEvent.click(screen.getByText("NOVA"));
  expect(onSelect).toHaveBeenCalled();
});
