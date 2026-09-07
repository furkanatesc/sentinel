"use client";
import { toast } from "sonner";
import type { NotificationConfig, SlackConnectionState } from "@/lib/api/types";

const STATE_META: Record<SlackConnectionState, { label: string; color: string }> = {
  connected: { label: "Bağlı", color: "#2FD98B" },
  disconnected: { label: "Bağlı değil", color: "#8A94A6" },
  error: { label: "Hata", color: "#F0476B" },
};

// SlackConnectionCard, webhook bağlantı durumu + workspace/channel + test bildirimi (simüle).
export default function SlackConnectionCard({ config }: { config: NotificationConfig }) {
  const meta = STATE_META[config.slackState];
  return (
    <div className="rounded-lg border border-border/60 bg-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full" style={{ background: meta.color }} aria-hidden />
            <span className="text-sm font-medium text-foreground">Slack — {meta.label}</span>
          </div>
          <p className="mt-1 text-xs text-foreground/60">
            {config.workspace} · <span className="font-mono text-foreground/80">{config.channel}</span>
          </p>
        </div>
        <button
          type="button"
          onClick={() => toast.success("Test bildirimi gönderildi (simüle)")}
          className="shrink-0 rounded-md border border-border/60 px-2.5 py-1 text-xs text-foreground hover:bg-muted"
        >
          Test bildirimi gönder
        </button>
      </div>
    </div>
  );
}
