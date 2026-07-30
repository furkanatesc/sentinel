import { EVENT_TYPE_DEFS, EVENT_SEVERITY } from "@/lib/feed/event-defs";
import { severityMeta } from "@/lib/format";
import type { EventType } from "@/lib/api/types";

const DEF_BY_KEY = Object.fromEntries(EVENT_TYPE_DEFS.map((d) => [d.key, d])) as Record<EventType, typeof EVENT_TYPE_DEFS[number]>;

export function EventTypeBadge({ type }: { type: EventType }) {
  const def = DEF_BY_KEY[type];
  const color = severityMeta[EVENT_SEVERITY[type]].color;
  const Icon = def.icon;
  return (
    <span className="inline-flex items-center gap-1.5 rounded px-1.5 py-0.5" style={{ color, backgroundColor: `${color}1f`, fontSize: 11, fontWeight: 600 }}>
      <Icon size={12} /> {def.label}
    </span>
  );
}
