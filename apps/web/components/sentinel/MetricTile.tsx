export function MetricTile({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-lg border border-border bg-surface-2 p-3">
      <div className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</div>
      <div className="mt-1 font-mono tabular-nums" style={{ fontSize: 18, fontWeight: 600 }}>{value}</div>
      {hint && <div className="text-muted-foreground" style={{ fontSize: 10 }}>{hint}</div>}
    </div>
  );
}
