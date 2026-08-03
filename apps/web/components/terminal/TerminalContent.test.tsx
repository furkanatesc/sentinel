import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { TerminalContent } from "./TerminalContent";

vi.mock("lightweight-charts", () => ({
  ColorType: { Solid: "solid" },
  createChart: vi.fn(() => ({
    addCandlestickSeries: vi.fn(() => ({ setData: vi.fn() })),
    timeScale: vi.fn(() => ({ fitContent: vi.fn() })),
    remove: vi.fn(),
  })),
}));

test("renders the four terminal regions", async () => {
  render(<QueryClientProvider client={getQueryClient()}><TerminalContent /></QueryClientProvider>);
  expect(screen.getByRole("heading", { name: "Terminal" })).toBeInTheDocument();
  await waitFor(() => expect(screen.getByText("Tokenlar")).toBeInTheDocument());   // watchlist
  await waitFor(() => expect(screen.getByText("PULSE")).toBeInTheDocument());      // market header
  expect(screen.getByRole("tab", { name: "Pozisyonlar" })).toBeInTheDocument();    // bottom tabs
});
