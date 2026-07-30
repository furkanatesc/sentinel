import type { RiskGroups, RiskItem } from "@/lib/api/types";
import { riskSeverityMeta } from "@/lib/format";

const CATS: { key: keyof RiskGroups; label: string }[] = [
  { key: "contract", label: "Kontrat Riski" },
  { key: "market", label: "Piyasa Riski" },
  { key: "creator", label: "Üretici Riski" },
];

function RiskRow({ r }: { r: RiskItem }) {
  const meta = riskSeverityMeta[r.severity];
  return (
    <li className="rounded-md border border-border bg-surface-2 p-3">
      <div className="flex items-center justify-between gap-2">
        <span style={{ fontSize: 13, fontWeight: 500 }}>{r.title}</span>
        <span className="rounded px-1.5 py-0.5" style={{ color: meta.color, backgroundColor: meta.bg, fontSize: 10, fontWeight: 600 }}>{meta.label}</span>
      </div>
      <div className="mt-1 text-muted-foreground" style={{ fontSize: 12 }}>{r.description}</div>
      {r.evidence && <div className="mt-1 font-mono text-muted-foreground" style={{ fontSize: 11 }}>Kanıt: {r.evidence}</div>}
      <div className="mt-1 text-muted-foreground" style={{ fontSize: 10 }}>İlk: {r.firstSeen} · Son: {r.lastSeen}</div>
    </li>
  );
}

export function RiskAnalysisTab({ risks }: { risks: RiskGroups }) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      {CATS.map((c) => (
        <div key={c.key}>
          <h3 className="mb-2">{c.label}</h3>
          {risks[c.key].length ? (
            <ul className="space-y-2">{risks[c.key].map((r) => <RiskRow key={r.id} r={r} />)}</ul>
          ) : (
            <div className="rounded-md border border-dashed border-border p-4 text-center text-muted-foreground" style={{ fontSize: 12 }}>Bu kategoride risk yok</div>
          )}
        </div>
      ))}
    </div>
  );
}
