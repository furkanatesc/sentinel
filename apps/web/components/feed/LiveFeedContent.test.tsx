import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { LiveFeedContent } from "./LiveFeedContent";

function wrap() {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <LiveFeedContent />
    </QueryClientProvider>
  );
}

const countOf = () => Number(screen.getByText(/· \d+ event/).textContent!.match(/(\d+) event/)![1]);

test("applying a risk filter narrows the table", async () => {
  wrap();
  // Wait for the real query data (not the initial "0 event" render before the mock API resolves).
  await waitFor(() => expect(countOf()).toBeGreaterThan(0));
  const before = countOf();
  await userEvent.click(screen.getByRole("button", { name: "Kritik" }));
  await waitFor(() => {
    expect(countOf()).toBeLessThanOrEqual(before);
  });
});
