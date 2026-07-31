import { STATUS_DEFS } from "@/lib/strategy/status-defs";
import type { StrategyStatus } from "@/lib/api/types";

export function StatusBadge({ status }: { status: StrategyStatus }) {
  const d = STATUS_DEFS[status];
  return (
    <span
      className="inline-flex items-center rounded-md px-2 py-0.5 font-medium"
      style={{ fontSize: 11, color: d.color, background: `${d.color}1A`, border: `1px solid ${d.color}40` }}
    >
      {d.label}
    </span>
  );
}
