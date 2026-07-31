"use client";
import Link from "next/link";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { Position } from "@/lib/api/types";
import { riskMeta } from "@/lib/format";
import { pnlColor } from "@/lib/position/risk-filter";
import { PositionActions } from "./PositionActions";

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="flex items-center justify-between border-b border-border py-2"><span className="text-muted-foreground" style={{ fontSize: 12 }}>{label}</span><span style={{ fontSize: 13 }}>{children}</span></div>;
}

export function PositionDetailDrawer({ position, onClose }: { position: Position | null; onClose: () => void }) {
  return (
    <Sheet open={!!position} onOpenChange={(o) => { if (!o) onClose(); }}>
      <SheetContent side="right" className="w-[380px] bg-popover">
        {position && (
          <>
            <SheetHeader><SheetTitle><span className="font-mono">{position.tokenSymbol}</span> · {position.strategyName}</SheetTitle></SheetHeader>
            <div className="mt-3 space-y-1">
              <Row label="Giriş / Güncel"><span className="font-mono">{position.entryPrice} / {position.currentPrice}</span></Row>
              <Row label="Boyut"><span className="font-mono">{position.sizeSol} SOL</span></Row>
              <Row label="PnL"><span className="font-mono" style={{ color: pnlColor(position.pnlSol) }}>{position.pnlSol} SOL (%{position.pnlPct})</span></Row>
              <Row label="Stop-Loss / Take-Profit"><span className="font-mono">%{position.stopLossPct} / %{position.takeProfitPct}</span></Row>
              <Row label="Token Risk"><span style={{ color: riskMeta[position.tokenRisk].color }}>{riskMeta[position.tokenRisk].label}</span></Row>
              <Row label="Creator Risk"><span style={{ color: riskMeta[position.creatorRisk].color }}>{riskMeta[position.creatorRisk].label}</span></Row>
              <Row label="Açılış">{position.openedAt}</Row>
            </div>
            <div className="mt-4"><PositionActions symbol={position.tokenSymbol} /></div>
            <Link href={`/tokens/${position.tokenSymbol}`} className="mt-4 inline-block rounded-md bg-primary px-4 py-2 text-primary-foreground" style={{ fontSize: 13, fontWeight: 500 }}>Token Detayına Git</Link>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
