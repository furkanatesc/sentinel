import { render, screen } from "@testing-library/react";
import { VersionHistory } from "./VersionHistory";
import { AuditLog } from "./AuditLog";
import type { StrategyVersion, AuditEntry } from "@/lib/api/types";

const versions: StrategyVersion[] = [{ version: "v1.3", date: "2026-07-28", note: "Stop-loss sıkılaştırıldı" }];
const audit: AuditEntry[] = [{ time: "2026-07-30 14:22", action: "Duraklatıldı", detail: "Drawdown eşiği" }];

test("version history lists versions with notes", () => {
  render(<VersionHistory versions={versions} />);
  expect(screen.getByText("v1.3")).toBeInTheDocument();
  expect(screen.getByText("Stop-loss sıkılaştırıldı")).toBeInTheDocument();
});

test("audit log lists actions with details", () => {
  render(<AuditLog audit={audit} />);
  expect(screen.getByText("Duraklatıldı")).toBeInTheDocument();
  expect(screen.getByText("Drawdown eşiği")).toBeInTheDocument();
});
