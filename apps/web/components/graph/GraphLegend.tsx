import { NODE_TYPE_DEFS, EDGE_TYPE_DEFS } from "@/lib/graph/graph-defs";

export function GraphLegend() {
  return (
    <div className="rounded-lg border border-border bg-card p-3" style={{ fontSize: 11 }}>
      <div className="mb-2 flex flex-wrap gap-3">
        <span className="text-muted-foreground">Düğümler:</span>
        {NODE_TYPE_DEFS.map((d) => (
          <span key={d.key} className="flex items-center gap-1.5"><span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: d.color }} />{d.label}</span>
        ))}
      </div>
      <div className="flex flex-wrap gap-3">
        <span className="text-muted-foreground">İlişkiler:</span>
        {EDGE_TYPE_DEFS.map((d) => (
          <span key={d.key} className="flex items-center gap-1.5"><span className="h-0.5 w-4" style={{ backgroundColor: d.color }} />{d.label}</span>
        ))}
      </div>
    </div>
  );
}
