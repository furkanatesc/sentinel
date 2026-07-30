// RiskAnalysisTab.test.tsx — kategoriler, severity ve boş durum
import { render, screen } from "@testing-library/react";
import { RiskAnalysisTab } from "./RiskAnalysisTab";
import type { RiskGroups } from "@/lib/api/types";

const risks: RiskGroups = {
  contract: [{ id: "c1", title: "Mint authority aktif", severity: "critical", description: "d", firstSeen: "12dk önce", lastSeen: "az önce" }],
  market: [], creator: [],
};

test("renders categories, severity, and empty state", () => {
  render(<RiskAnalysisTab risks={risks} />);
  expect(screen.getByText("Kontrat Riski")).toBeInTheDocument();
  expect(screen.getByText("Mint authority aktif")).toBeInTheDocument();
  expect(screen.getByText("Kritik")).toBeInTheDocument();
  expect(screen.getAllByText("Bu kategoride risk yok").length).toBe(2); // market + creator
});
