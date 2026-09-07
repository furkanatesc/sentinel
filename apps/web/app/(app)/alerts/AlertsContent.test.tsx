import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertsContent from "./AlertsContent";

vi.mock("@/components/alerts/AlertRulesPanel", () => ({
  default: ({ onNew }: { onNew: () => void }) => <button onClick={onNew}>rules-panel</button>,
}));
vi.mock("@/components/alerts/AlertHistoryPanel", () => ({ default: () => <div>history-panel</div> }));
vi.mock("@/components/alerts/AlertRuleForm", () => ({ default: () => <div>rule-form</div> }));

describe("AlertsContent", () => {
  it("sekmeler arası geçiş", () => {
    render(<AlertsContent />);
    expect(screen.getByText("rules-panel")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: /geçmiş/i }));
    expect(screen.getByText("history-panel")).toBeInTheDocument();
  });
  it("Yeni Kural formu açar", () => {
    render(<AlertsContent />);
    fireEvent.click(screen.getByText("rules-panel"));
    expect(screen.getByText("rule-form")).toBeInTheDocument();
  });
});
