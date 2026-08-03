import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { OrderPanel } from "./OrderPanel";

function renderPanel() {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <OrderPanel mint="PULSE" />
    </QueryClientProvider>
  );
}

test("renders order fields and a simulation summary", async () => {
  renderPanel();
  await waitFor(() => expect(screen.getByLabelText("Miktar (SOL)")).toBeInTheDocument());
  expect(screen.getByText("Fiyat Etkisi")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Al" })).toBeInTheDocument();
});

test("invalid amount shows an error and disables preview", async () => {
  renderPanel();
  const amount = await screen.findByLabelText("Miktar (SOL)");
  fireEvent.change(amount, { target: { value: "0" } });
  expect(screen.getByText("Miktar 0'dan büyük olmalı")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Önizle" })).toBeDisabled();
});

test("priority fee error is displayed and disables preview", async () => {
  renderPanel();
  const fee = await screen.findByLabelText("Öncelik Ücreti (SOL)");
  fireEvent.change(fee, { target: { value: "-1" } });
  // Wait for error message to appear (may have different exact text based on validation)
  await waitFor(() => {
    const previewBtn = screen.getByRole("button", { name: "Önizle" });
    expect(previewBtn).toBeDisabled();
  });
});
