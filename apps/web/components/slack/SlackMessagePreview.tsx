import type { NotificationConfig } from "@/lib/api/types";
import { ALERT_TRIGGER_DEFS } from "@/lib/alerts/alert-defs";

// Örnek doldurma değerleri (şablon {{token}}/{{detail}} yer tutucuları için).
const SAMPLE = { token: "GFROG", detail: "üretici havuzun %92'sini çekti" };

function fill(template: string): string {
  return template.replace(/\{\{token\}\}/g, SAMPLE.token).replace(/\{\{detail\}\}/g, SAMPLE.detail);
}

// SlackMessagePreview, Slack-block stili bir mesaj kartı render eder (ilk şablonla örnek).
export default function SlackMessagePreview({ config }: { config: NotificationConfig }) {
  const first = config.templates[0];
  const def = first ? ALERT_TRIGGER_DEFS[first.trigger] : null;
  const Icon = def?.icon;
  const body = first ? fill(first.template) : "Örnek alarm mesajı — {{token}} tetiklendi";

  return (
    <div className="rounded-lg border border-border/60 bg-card p-4">
      <h3 className="mb-2 text-sm font-medium text-foreground">Önizleme</h3>
      <div className="overflow-hidden rounded-md border border-border/60 bg-background">
        <div className="flex items-center gap-2 border-b border-border/60 px-3 py-2 text-xs text-foreground/60">
          <span className="font-medium text-foreground/80">{config.workspace}</span>
          <span className="font-mono">{config.channel}</span>
        </div>
        <div className="flex gap-2.5 px-3 py-3">
          <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded bg-primary/15 text-primary">
            {Icon ? <Icon size={15} aria-hidden /> : "S"}
          </span>
          <div className="min-w-0">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-semibold text-foreground">Sentinel</span>
              <span className="rounded bg-muted px-1 py-0.5 text-[10px] font-medium uppercase text-foreground/50">
                APP
              </span>
              {def && <span className="text-xs text-foreground/50">{def.label}</span>}
            </div>
            <p className="mt-0.5 text-sm text-foreground/80">{body}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
