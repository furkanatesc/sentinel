import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { BacktestParamsForm } from "./BacktestParamsForm";

function renderForm(onRun = () => {}) {
  return render(
    <QueryClientProvider client={getQueryClient()}>
      <BacktestParamsForm onRun={onRun} />
    </QueryClientProvider>
  );
}

test("renders fields and runs with valid params", async () => {
  const onRun = vi.fn();
  renderForm(onRun);
  await waitFor(() => expect(screen.getByLabelText("Başlangıç Sermayesi (SOL)")).toBeInTheDocument());
  fireEvent.click(screen.getByRole("button", { name: "Çalıştır" }));
  expect(onRun).toHaveBeenCalledTimes(1);
});

test("invalid capital blocks run and shows error", async () => {
  const onRun = vi.fn();
  renderForm(onRun);
  const capital = await screen.findByLabelText("Başlangıç Sermayesi (SOL)");
  fireEvent.change(capital, { target: { value: "0" } });
  fireEvent.click(screen.getByRole("button", { name: "Çalıştır" }));
  expect(screen.getByText("Sermaye 0'dan büyük olmalı")).toBeInTheDocument();
  expect(onRun).not.toHaveBeenCalled();
});
