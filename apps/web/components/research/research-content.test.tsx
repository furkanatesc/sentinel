import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ResearchContent } from "./ResearchContent";
import { useResearchStore } from "@/lib/store/research";

function renderWithQuery(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

beforeEach(() => useResearchStore.getState().reset());

describe("ResearchContent", () => {
  it("shows suggestions on empty thread and disclaimer", async () => {
    renderWithQuery(<ResearchContent />);
    expect(screen.getByText(/bilgilendirme/i)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: /GFROG neden riskli/ })).toBeInTheDocument());
  });

  it("sends a question via the composer and renders the streamed answer", async () => {
    renderWithQuery(<ResearchContent />);
    const box = screen.getByPlaceholderText(/soru/i);
    fireEvent.change(box, { target: { value: "GFROG neden riskli?" } });
    fireEvent.keyDown(box, { key: "Enter" });
    await waitFor(() => expect(screen.getByText(/GFROG neden riskli\?/)).toBeInTheDocument()); // user bubble
    await act(async () => { await new Promise((r) => setTimeout(r, 1500)); }); // let stream finish (real timers)
    await waitFor(() => expect(useResearchStore.getState().isStreaming).toBe(false));
    expect(useResearchStore.getState().messages.at(-1)!.status).toBe("done");
  });
});
