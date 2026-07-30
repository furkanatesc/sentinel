import { scoreToLevel, riskMeta } from "@/lib/format";

interface ScoreBadgeProps { score: number; label?: string; showBar?: boolean; size?: "sm" | "md"; }

export function ScoreBadge({ score, label, showBar = false, size = "sm" }: ScoreBadgeProps) {
  const level = scoreToLevel(score);
  const meta = riskMeta[level];
  return (
    <div className="inline-flex flex-col gap-1">
      <div
        className="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5"
        style={{ backgroundColor: meta.bg, border: `1px solid ${meta.border}` }}
        title={label ? `${label}: ${meta.label}` : meta.label}
      >
        <span className="font-mono tabular-nums" style={{ color: meta.color, fontSize: size === "sm" ? 12 : 14, fontWeight: 600 }}>
          {score}
        </span>
        <span style={{ color: meta.color, fontSize: 11, opacity: 0.85 }}>{meta.label}</span>
      </div>
      {showBar && (
        <div className="h-1 w-full overflow-hidden rounded-full bg-surface-3">
          <div className="h-full rounded-full" style={{ width: `${score}%`, backgroundColor: meta.color }} />
        </div>
      )}
    </div>
  );
}
