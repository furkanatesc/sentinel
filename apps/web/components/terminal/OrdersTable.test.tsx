import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { OrdersTable } from "./OrdersTable";

vi.mock("sonner", () => ({ toast: Object.assign(vi.fn(), { warning: vi.fn() }) }));
import { toast } from "sonner";

test("lists orders and cancels an open one with a simulate toast", async () => {
  render(<QueryClientProvider client={getQueryClient()}><OrdersTable /></QueryClientProvider>);
  await waitFor(() => expect(screen.getAllByText(/Al|Sat/).length).toBeGreaterThan(0));
  const cancelButtons = screen.queryAllByRole("button", { name: "İptal" });
  expect(cancelButtons.length).toBeGreaterThan(0);
  fireEvent.click(cancelButtons[0]);
  expect(toast).toHaveBeenCalled();
});
