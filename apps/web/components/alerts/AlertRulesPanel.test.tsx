import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertRulesPanel from "./AlertRulesPanel";

vi.mock("@/lib/hooks/queries", () => ({
  useAlertRules: () => ({
    data: [
      { id: "r1", name: "Kural Bir", trigger: "whale_activity", scope: "Tüm tokenlar", minLiquidity: 0, minCreatorScore: 0, maxRisk: "high", channels: ["slack"], enabled: true },
    ],
    isLoading: false,
    isError: false,
  }),
}));

describe("AlertRulesPanel", () => {
  it("kuralları + trigger etiketini gösterir", () => {
    render(<AlertRulesPanel />);
    expect(screen.getByText("Kural Bir")).toBeInTheDocument();
    expect(screen.getByText("Balina Hareketi")).toBeInTheDocument();
  });
  it("toggle local state'i çevirir", () => {
    render(<AlertRulesPanel />);
    const sw = screen.getByRole("switch");
    expect(sw).toBeChecked();
    fireEvent.click(sw);
    expect(sw).not.toBeChecked();
  });
});
