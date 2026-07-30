import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { WalletGraphContent } from "./WalletGraphContent";
vi.mock("cytoscape", () => ({ default: vi.fn(() => ({ on: vi.fn(), destroy: vi.fn() })) }));
test("renders title, filters, legend and empty detail hint", async () => {
  render(<QueryClientProvider client={getQueryClient()}><WalletGraphContent /></QueryClientProvider>);
  expect(screen.getByRole("heading", { name: "Cüzdan Grafiği" })).toBeInTheDocument();
  expect(screen.getByText("Detay için bir düğüm seç")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByText("Creator Cüzdanı")).toBeInTheDocument()); // legend
});
