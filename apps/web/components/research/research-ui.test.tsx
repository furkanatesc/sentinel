import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SourceChip } from "./SourceChip";
import { SuggestionChips } from "./SuggestionChips";
import { ChatMessageBubble } from "./ChatMessageBubble";
import { InfoDisclaimerBanner } from "./InfoDisclaimerBanner";

describe("SourceChip", () => {
  it("renders a link for token sources", () => {
    render(<SourceChip source={{ id: "s1", kind: "token", label: "AERO", ref: "AERO" }} />);
    expect(screen.getByRole("link", { name: /AERO/ })).toHaveAttribute("href", "/tokens/AERO");
  });
  it("renders a non-link chip for tx sources", () => {
    render(<SourceChip source={{ id: "s2", kind: "tx", label: "5xA…" }} />);
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText(/5xA/)).toBeInTheDocument();
  });
});

describe("SuggestionChips", () => {
  it("fires onPick with the suggestion text", () => {
    const onPick = vi.fn();
    render(<SuggestionChips suggestions={[{ id: "q1", text: "GFROG neden riskli?" }]} onPick={onPick} disabled={false} />);
    fireEvent.click(screen.getByRole("button", { name: /GFROG/ }));
    expect(onPick).toHaveBeenCalledWith("GFROG neden riskli?");
  });
  it("disables chips while streaming", () => {
    render(<SuggestionChips suggestions={[{ id: "q1", text: "x" }]} onPick={() => {}} disabled />);
    expect(screen.getByRole("button", { name: "x" })).toBeDisabled();
  });
});

describe("ChatMessageBubble", () => {
  it("shows a streaming cursor while streaming", () => {
    render(<ChatMessageBubble message={{ id: "a1", role: "assistant", text: "yaz", status: "streaming" }} />);
    expect(screen.getByTestId("stream-cursor")).toBeInTheDocument();
  });
  it("renders sources when done", () => {
    render(<ChatMessageBubble message={{ id: "a2", role: "assistant", text: "bitti", status: "done",
      sources: [{ id: "s1", kind: "token", label: "AERO", ref: "AERO" }] }} />);
    expect(screen.getByRole("link", { name: /AERO/ })).toBeInTheDocument();
  });
});

describe("InfoDisclaimerBanner", () => {
  it("renders the informational-analysis label", () => {
    render(<InfoDisclaimerBanner />);
    expect(screen.getByText(/bilgilendirme/i)).toBeInTheDocument();
  });
});
