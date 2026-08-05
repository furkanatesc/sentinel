import { scoreToLevel, riskMeta } from "@/lib/format";

interface ScoreBadgeProps { score: number; label?: string; showBar?: boolean; size?: "sm" | "md"; }

// 1a'da backend henüz skor üretmediğinde creatorScore/safetyScore tam olarak 0 gelir.
// Bu, "en kötü/kritik" anlamına gelmez — "henüz hesaplanmadı" demektir. 0'ı sahte bir
// kritik skor gibi göstermek yerine dürüst nötr bir rozet gösteriyoruz. Mock veride skor
// hiç 0 olmadığından bu dal yalnız gerçek backend'in henüz-hesaplamadığı skorları etkiler.
const NEUTRAL_META = { label: "Henüz yok", color: "#8A94A6", bg: "rgba(138,148,166,0.12)", border: "rgba(138,148,166,0.30)" };

export function ScoreBadge({ score, label, showBar = false, size = "sm" }: ScoreBadgeProps) {
  const isNeutral = score === 0;
  const level = scoreToLevel(score);
  const meta = isNeutral ? NEUTRAL_META : riskMeta[level];
  return (
    <div className="inline-flex flex-col gap-1">
      <div
        className="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5"
        style={{ backgroundColor: meta.bg, border: `1px solid ${meta.border}` }}
        title={label ? `${label}: ${meta.label}` : meta.label}
      >
        <span className="font-mono tabular-nums" style={{ color: meta.color, fontSize: size === "sm" ? 12 : 14, fontWeight: 600 }}>
          {isNeutral ? "—" : score}
        </span>
        <span style={{ color: meta.color, fontSize: 11, opacity: 0.85 }}>{meta.label}</span>
      </div>
      {showBar && (
        <div className="h-1 w-full overflow-hidden rounded-full bg-surface-3">
          <div className="h-full rounded-full" style={{ width: `${isNeutral ? 0 : score}%`, backgroundColor: meta.color }} />
        </div>
      )}
    </div>
  );
}
