import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { BottomTabsPanel } from "./BottomTabsPanel";

function renderPanel() {
  return render(<QueryClientProvider client={getQueryClient()}><BottomTabsPanel /></QueryClientProvider>);
}

test("shows tab triggers and switches to the Orders tab", async () => {
  renderPanel();
  expect(screen.getByRole("tab", { name: "Pozisyonlar" })).toBeInTheDocument();
  fireEvent.click(screen.getByRole("tab", { name: "Emirler" }));
  await waitFor(() => expect(screen.getByText("Yön")).toBeInTheDocument());
});
