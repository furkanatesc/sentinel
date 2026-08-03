"use client";
import { toast } from "sonner";
import { useSessionStore } from "@/lib/store/session";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { simulateOrder } from "@/lib/terminal/order-logic";
import { ORDER_SIDE_DEFS, MOCK_WALLET_SOL, type OrderDraft } from "@/lib/terminal/order-defs";
import { scoreToLevel, riskMeta } from "@/lib/format";
import type { MarketData } from "@/lib/api/types";

function Row({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="flex items-center justify-between py-1" style={{ fontSize: 13 }}>
      <span className="text-muted-foreground">{label}</span>
      <span style={{ color }}>{value}</span>
    </div>
  );
}

export function OrderConfirmDialog({ open, draft, market, onClose }: {
  open: boolean; draft: OrderDraft; market: MarketData; onClose: () => void;
}) {
  const mode = useSessionStore((s) => s.tradingMode);
  const sim = simulateOrder(draft, market);
  const side = ORDER_SIDE_DEFS.find((s) => s.key === draft.side)!;
  const risky = market.tokenScore < 50 || market.creatorScore < 50;

  const confirm = () => {
    const summary = `${side.label} ${draft.amountSol} SOL · ${market.symbol}`;
    if (mode === "live") {
      toast.warning(`CANLI mod — ${summary}`, { description: "Gerçek para. Bu demoda emir gönderilmez." });
    } else {
      toast(`Emir simüle edildi — ${summary}`, {
        description: `${mode === "paper" ? "Kağıt" : "Gölge"} modda. Etki %${sim.priceImpactPct}, min ${sim.minReceived}.`,
      });
    }
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader><DialogTitle>Emir Onayı</DialogTitle></DialogHeader>
        <div>
          <Row label="Token" value={market.symbol} />
          <Row label="İşlem" value={side.label} color={side.color} />
          <Row label="Tutar" value={`${draft.amountSol} SOL`} />
          <Row label="Tahmini Fiyat" value={`${sim.estPrice} SOL`} />
          <Row label="Slippage" value={`%${draft.slippagePct}`} />
          <Row label="Fiyat Etkisi" value={`%${sim.priceImpactPct}`} />
          <Row label="Min. Alınan" value={`${sim.minReceived}`} />
          <Row label="Tahmini Ücret" value={`${sim.estFeeSol} SOL`} />
          <Row label="Cüzdan Bakiyesi" value={`${MOCK_WALLET_SOL} SOL`} />
          <Row label="Token Skoru" value={`${market.tokenScore} · ${riskMeta[scoreToLevel(market.tokenScore)].label}`} color={riskMeta[scoreToLevel(market.tokenScore)].color} />
          <Row label="Creator Skoru" value={`${market.creatorScore} · ${riskMeta[scoreToLevel(market.creatorScore)].label}`} color={riskMeta[scoreToLevel(market.creatorScore)].color} />
        </div>
        {risky && (
          <div className="mt-2 rounded-md px-3 py-2" style={{ fontSize: 12, color: "#FFB020", backgroundColor: "rgba(255,176,32,0.1)", border: "1px solid rgba(255,176,32,0.35)" }}>
            Düşük token/creator skoru — yüksek risk.
          </div>
        )}
        {mode === "live" && (
          <div className="mt-2 rounded-md px-3 py-2" style={{ fontSize: 12, color: "#F0476B", backgroundColor: "rgba(240,71,107,0.12)", border: "1px solid rgba(240,71,107,0.4)" }}>
            CANLI işlem — gerçek para riski. Bu demoda emir gönderilmez.
          </div>
        )}
        <DialogFooter>
          <button onClick={onClose} className="rounded-md border border-border px-3 py-1.5" style={{ fontSize: 12 }}>Vazgeç</button>
          <button onClick={confirm} className="rounded-md px-3 py-1.5 text-primary-foreground" style={{ fontSize: 12, fontWeight: 600, backgroundColor: side.color, color: "#08210F" }}>Onayla</button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
