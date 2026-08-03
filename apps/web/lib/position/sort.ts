import type { Position } from "@/lib/api/types";
import type { SortKey } from "@/components/position/PositionsTable";

// Shared by BottomTabsPanel (terminal) and PositionsContent (positions page) so both
// surfaces sort rows identically instead of maintaining two copies of the same logic.
export function sortPositions(rows: Position[], key: SortKey): Position[] {
  return [...rows].sort((a, b) => {
    if (key === "ageLabel") return parseInt(b.ageLabel, 10) - parseInt(a.ageLabel, 10);
    return (b[key] as number) - (a[key] as number);
  });
}
