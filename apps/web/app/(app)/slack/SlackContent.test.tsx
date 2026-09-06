import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import SlackContent from "./SlackContent";

const toast = vi.fn();
vi.mock("sonner", () => ({
  toast: Object.assign((...a: unknown[]) => toast(...a), { success: (...a: unknown[]) => toast(...a) }),
}));
vi.mock("@/lib/hooks/queries", () => ({
  useNotificationConfig: () => ({
    data: {
      slackState: "connected",
      channel: "#alerts",
      workspace: "Sentinel HQ",
      minSeverity: "warning",
      quietHours: { start: "23:00", end: "07:00", enabled: false },
      templates: [{ trigger: "new_mint", template: "✨ {{token}}" }],
      tradeApproval: true,
    },
    isLoading: false,
    isError: false,
  }),
}));

describe("SlackContent", () => {
  it("bağlantı + channel gösterir", () => {
    render(<SlackContent />);
    // #alerts hem bağlantı kartında hem mesaj önizlemesinde görünür.
    expect(screen.getAllByText("#alerts").length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Sentinel HQ/).length).toBeGreaterThan(0);
  });
  it("test bildirimi → simüle toast", () => {
    render(<SlackContent />);
    fireEvent.click(screen.getByRole("button", { name: /test bildirimi/i }));
    expect(toast).toHaveBeenCalled();
  });
});
