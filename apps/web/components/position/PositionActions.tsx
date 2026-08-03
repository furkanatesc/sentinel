"use client";
import { toast } from "sonner";
import { useSessionStore } from "@/lib/store/session";

export function PositionActions({ symbol }: { symbol: string }) {
  const mode = useSessionStore((s) => s.tradingMode);
  const act = (label: string) => {
    if (mode === "live") toast.warning(`CANLI mod — ${label} ${symbol}`, { description: "Gerçek para. Bu demoda emir gönderilmez." });
    else toast(`${label} ${symbol}`, { description: `${mode === "paper" ? "Kağıt" : "Gölge"} modda simüle edilir.` });
  };
  return (
    <div className="flex items-center gap-1.5">
      <button onClick={() => act("Kapat")} className="rounded-md px-2 py-1" style={{ backgroundColor: "rgba(240,71,107,0.15)", border: "1px solid rgba(240,71,107,0.4)", color: "#F0476B", fontSize: 11, fontWeight: 600 }}>Kapat</button>
      <button onClick={() => act("SL/TP ayarla")} className="rounded-md border border-border px-2 py-1 hover:bg-accent" style={{ fontSize: 11 }}>SL/TP</button>
    </div>
  );
}
