import type { NotificationConfig } from "@/lib/api/types";
import { ALERT_TRIGGER_DEFS } from "@/lib/alerts/alert-defs";

// AlertTemplateList, tetikleyici-başına Slack mesaj şablonlarını gösterir (read-only).
export default function AlertTemplateList({ templates }: { templates: NotificationConfig["templates"] }) {
  return (
    <div className="rounded-lg border border-border/60 bg-card p-4">
      <h3 className="mb-2 text-sm font-medium text-foreground">Mesaj şablonları</h3>
      {templates.length === 0 ? (
        <p className="text-xs text-foreground/50">Şablon yok.</p>
      ) : (
        <ul className="space-y-2">
          {templates.map((t) => {
            const def = ALERT_TRIGGER_DEFS[t.trigger];
            const Icon = def.icon;
            return (
              <li key={t.trigger} className="text-xs">
                <span className="flex items-center gap-1.5 text-foreground/70">
                  <Icon size={12} aria-hidden />
                  {def.label}
                </span>
                <code className="mt-0.5 block rounded bg-muted/50 px-2 py-1 font-mono text-[11px] text-foreground/80">
                  {t.template}
                </code>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
