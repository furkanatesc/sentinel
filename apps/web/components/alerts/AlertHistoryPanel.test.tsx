import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertHistoryPanel from "./AlertHistoryPanel";

vi.mock("@/lib/hooks/queries", () => ({
  useAlerts: () => ({
    data: [
      { id: "a1", type: "Balina Alımı", token: "PULSE", detail: "x", severity: "positive", time: "az önce" },
      { id: "a2", type: "Likidite Çekildi", token: "GFROG", detail: "y", severity: "critical", time: "18sn önce" },
    ],
    isLoading: false,
    isError: false,
  }),
}));

describe("AlertHistoryPanel", () => {
  it("alarmları listeler", () => {
    render(<AlertHistoryPanel />);
    expect(screen.getByText("PULSE")).toBeInTheDocument();
    expect(screen.getByText("GFROG")).toBeInTheDocument();
  });
  it("severity filtresi daraltır", () => {
    render(<AlertHistoryPanel />);
    fireEvent.click(screen.getByRole("button", { name: /kritik/i }));
    expect(screen.queryByText("PULSE")).not.toBeInTheDocument();
    expect(screen.getByText("GFROG")).toBeInTheDocument();
  });
});
