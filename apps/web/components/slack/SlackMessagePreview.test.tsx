import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import SlackMessagePreview from "./SlackMessagePreview";
import type { NotificationConfig } from "@/lib/api/types";

const config: NotificationConfig = {
  slackState: "connected",
  channel: "#alerts",
  workspace: "Sentinel HQ",
  minSeverity: "warning",
  quietHours: { start: "23:00", end: "07:00", enabled: false },
  templates: [{ trigger: "liquidity_removed", template: "🚨 {{token}}: {{detail}}" }],
  tradeApproval: true,
};

describe("SlackMessagePreview", () => {
  it("channel + örnek mesaj render eder", () => {
    render(<SlackMessagePreview config={config} />);
    expect(screen.getByText("#alerts")).toBeInTheDocument();
    expect(screen.getByText(/GFROG/)).toBeInTheDocument();
    expect(screen.getByText(/Likidite Çekildi/)).toBeInTheDocument();
  });

  it("şablon yoksa çökme yok", () => {
    render(<SlackMessagePreview config={{ ...config, templates: [] }} />);
    expect(screen.getByText("#alerts")).toBeInTheDocument();
  });
});
