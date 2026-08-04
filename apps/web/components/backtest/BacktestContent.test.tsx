import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { BacktestContent } from "./BacktestContent";

function renderContent() {
  return render(<QueryClientProvider client={getQueryClient()}><BacktestContent /></QueryClientProvider>);
}

test("shows empty state before a run, then results after Çalıştır", async () => {
  renderContent();
  expect(screen.getByRole("heading", { name: "Geriye Test" })).toBeInTheDocument();
  expect(screen.getByText("Parametreleri seç ve Çalıştır'a bas")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByLabelText("Strateji")).toBeInTheDocument());
  fireEvent.click(screen.getByRole("button", { name: "Çalıştır" }));
  await waitFor(() => expect(screen.getByText("Net PnL (SOL)")).toBeInTheDocument());
  expect(screen.getByText("Sermaye Eğrisi")).toBeInTheDocument();
});
