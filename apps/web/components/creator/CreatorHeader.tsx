"use client";
import { Star, Send } from "lucide-react";
import { toast } from "sonner";
import type { CreatorProfile } from "@/lib/api/types";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { riskMeta } from "@/lib/format";

export function CreatorHeader({ profile }: { profile: CreatorProfile }) {
  const rm = riskMeta[profile.riskLevel];
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <h1 style={{ fontSize: 18 }}>Üretici</h1>
            <span className="rounded px-1.5 py-0.5" style={{ color: rm.color, backgroundColor: rm.bg, fontSize: 11, fontWeight: 600 }}>{rm.label}</span>
          </div>
          <WalletAddress address={profile.address} />
          <span className="text-muted-foreground" style={{ fontSize: 11 }}>Cüzdan yaşı: {profile.walletAgeDays} gün · İlk görülme: {profile.firstSeen}</span>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => toast("Üretici izleniyor")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Star size={14} /> İzle</button>
          <button onClick={() => toast("Telegram alarmı oluşturuldu")} className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 hover:bg-accent" style={{ fontSize: 12 }}><Send size={14} /> Telegram Alarmı</button>
        </div>
      </div>
    </div>
  );
}
