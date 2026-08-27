import type { WorkerState } from "@/lib/api/types";

export const WORKER_STATE_DEFS: Record<WorkerState, { label: string; color: string }> = {
  ok: { label: "Çalışıyor", color: "#2FD98B" },
  starting: { label: "Başlıyor", color: "#3E9BFF" },
  degraded: { label: "Bozulmuş", color: "#FFB020" },
  stalled: { label: "Durdu", color: "#F0476B" },
  off: { label: "Kapalı", color: "#8A94A6" },
};
