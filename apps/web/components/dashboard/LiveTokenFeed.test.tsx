import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { LiveTokenFeed } from "./LiveTokenFeed";

function wrap() {
  const client = getQueryClient();
  return render(<QueryClientProvider client={client}><LiveTokenFeed /></QueryClientProvider>);
}

test("renders rows and marks <60s tokens fresh", async () => {
  wrap();
  await waitFor(() => expect(screen.getByText("SolPulse")).toBeInTheDocument());
  const gfrogRow = screen.getByText("GigaFrog").closest("tr")!;
  expect(gfrogRow.getAttribute("data-fresh")).toBe("true"); // 12s old
});

test("sorting by liquidity puts highest first", async () => {
  wrap();
  await waitFor(() => expect(screen.getByText("SolPulse")).toBeInTheDocument());
  await userEvent.click(screen.getByRole("button", { name: /Lik/ }));
  const firstRow = screen.getAllByRole("row")[1];
  expect(within(firstRow).getByText("Helios")).toBeInTheDocument(); // $320K liquidity
});
