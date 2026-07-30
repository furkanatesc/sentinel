"use client";
import Link from "next/link";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import type { FeedEvent } from "@/lib/api/types";
import { formatAge, formatUsd, formatPct, riskMeta } from "@/lib/format";
import { EventTypeBadge } from "./EventTypeBadge";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="flex items-center justify-between border-b border-border py-2"><span className="text-muted-foreground" style={{ fontSize: 12 }}>{label}</span><span style={{ fontSize: 13 }}>{children}</span></div>;
}

export function EventDetailDrawer({ event, onClose }: { event: FeedEvent | null; onClose: () => void }) {
  return (
    <Sheet open={!!event} onOpenChange={(o) => { if (!o) onClose(); }}>
      <SheetContent side="right" className="w-[380px] bg-popover">
        {event && (
          <>
            <SheetHeader><SheetTitle><EventTypeBadge type={event.type} /></SheetTitle></SheetHeader>
            <div className="mt-3 space-y-1">
              <Row label="Token"><span className="font-mono">{event.symbol}</span></Row>
              <Row label="Mint"><WalletAddress address={event.mint} /></Row>
              <Row label="Kaynak">{event.launchpad} · {event.dex}</Row>
              <Row label="Likidite"><span className="font-mono">{formatUsd(event.liquidity)}</span></Row>
              <Row label="Creator"><ScoreBadge score={event.creatorScore} /></Row>
              <Row label="Risk"><span style={{ color: riskMeta[event.riskLevel].color }}>{riskMeta[event.riskLevel].label}</span></Row>
              <Row label="Yaş"><span className="font-mono">{formatAge(event.tokenAgeSeconds)}</span></Row>
              <Row label="5dk Hacim"><span className="font-mono">{formatUsd(event.volume5m)}</span></Row>
              <Row label="Holder Büyümesi"><span className="font-mono">{formatPct(event.holderGrowthPct)}</span></Row>
              <Row label="Zaman">{event.time}</Row>
            </div>
            <p className="mt-3 text-muted-foreground" style={{ fontSize: 12 }}>{event.detail}</p>
            <Link href={`/tokens/${event.symbol}`} className="mt-4 inline-block rounded-md bg-primary px-4 py-2 text-primary-foreground" style={{ fontSize: 13, fontWeight: 500 }}>
              Token Detayına Git
            </Link>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
