import {
  Sparkles, FileText, Droplet, ArrowLeftRight, PlusCircle, MinusCircle,
  TrendingDown, Fish, ShieldAlert, Activity, Zap, type LucideIcon,
} from "lucide-react";
import type { EventType } from "@/lib/api/types";
import type { AlertSeverity } from "@/lib/format";

export const EVENT_SEVERITY: Record<EventType, AlertSeverity> = {
  new_mint: "info", metadata_created: "info", pool_created: "info", first_swap: "positive",
  liquidity_added: "positive", liquidity_removed: "critical", creator_sell: "warning",
  whale_buy: "positive", suspicious_cluster: "critical", score_change: "warning", strategy_signal: "info",
};

export interface EventTypeDef { key: EventType; label: string; icon: LucideIcon; }

export const EVENT_TYPE_DEFS: EventTypeDef[] = [
  { key: "new_mint", label: "Yeni Mint", icon: Sparkles },
  { key: "metadata_created", label: "Metadata Oluşturuldu", icon: FileText },
  { key: "pool_created", label: "Havuz Açıldı", icon: Droplet },
  { key: "first_swap", label: "İlk Swap", icon: ArrowLeftRight },
  { key: "liquidity_added", label: "Likidite Eklendi", icon: PlusCircle },
  { key: "liquidity_removed", label: "Likidite Çekildi", icon: MinusCircle },
  { key: "creator_sell", label: "Üretici Satışı", icon: TrendingDown },
  { key: "whale_buy", label: "Balina Alımı", icon: Fish },
  { key: "suspicious_cluster", label: "Şüpheli Küme", icon: ShieldAlert },
  { key: "score_change", label: "Skor Değişti", icon: Activity },
  { key: "strategy_signal", label: "Strateji Sinyali", icon: Zap },
];
