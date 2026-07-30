"use client";
import { Star, Send, FlaskConical } from "lucide-react";
import { toast } from "sonner";
import { useSessionStore } from "@/lib/store/session";

export function TokenActions({ symbol }: { symbol: string }) {
  const mode = useSessionStore((s) => s.tradingMode);
  const trade = (side: "Al" | "Sat") => {
    if (mode === "live") toast.warning(`CANLI mod — ${side} ${symbol}`, { description: "Gerçek para. Bu demoda emir gönderilmez." });
    else toast(`${side} ${symbol}`, { description: `${mode === "paper" ? "Kağıt" : "Gölge"} modda simüle edilir.` });
  };
  return (
    <div className="flex flex-wrap items-center gap-2">
      <button onClick={() => toast("İzleme listesine eklendi")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Star size={14} /> İzle</button>
      <button onClick={() => toast("Telegram alarmı oluşturuldu")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Send size={14} /> Telegram Alarmı</button>
      <button onClick={() => toast(`${symbol} işlemi simüle ediliyor…`)} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><FlaskConical size={14} /> Simüle Et</button>
      <button onClick={() => trade("Al")} className="rounded-md px-4 py-1.5 text-primary-foreground" style={{ backgroundColor: "#2FD98B", fontSize: 12, fontWeight: 600, color: "#08210F" }}>Al</button>
      <button onClick={() => trade("Sat")} className="rounded-md px-4 py-1.5" style={{ backgroundColor: "rgba(240,71,107,0.15)", border: "1px solid rgba(240,71,107,0.4)", color: "#F0476B", fontSize: 12, fontWeight: 600 }}>Sat</button>
    </div>
  );
}
