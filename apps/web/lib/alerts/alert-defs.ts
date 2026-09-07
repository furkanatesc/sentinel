import type { AlertTriggerType, DeliveryChannel } from "@/lib/api/types";
import type { RiskLevel, AlertSeverity } from "@/lib/format";
import {
  Sparkles, Droplet, DropletOff, TrendingDown, Fish, Users, Activity, Zap,
  Globe, Hash, Mail, Webhook, type LucideIcon,
} from "lucide-react";

// OCP: alarm tetikleyici registry'si — form select, kural kartı etiketi ve Slack önizleme
// buradan türer. Yeni bir trigger eklemek = tek satır.
export const ALERT_TRIGGER_DEFS: Record<AlertTriggerType, { label: string; icon: LucideIcon; description: string }> = {
  new_mint: { label: "Yeni Mint", icon: Sparkles, description: "Yeni token oluşturuldu" },
  liquidity_added: { label: "İlk Likidite", icon: Droplet, description: "Havuz likiditesi eklendi" },
  liquidity_removed: { label: "Likidite Çekildi", icon: DropletOff, description: "Havuzdan likidite çekildi" },
  creator_sale: { label: "Üretici Satışı", icon: TrendingDown, description: "Üretici token sattı" },
  whale_activity: { label: "Balina Hareketi", icon: Fish, description: "Büyük cüzdan işlemi" },
  holder_growth: { label: "Holder Artışı", icon: Users, description: "Holder sayısı hızlı arttı" },
  score_change: { label: "Skor Değişti", icon: Activity, description: "Güvenlik skoru değişti" },
  strategy_signal: { label: "Strateji Sinyali", icon: Zap, description: "Bir strateji sinyal üretti" },
};

// Kural "maks. risk" tavanı için anlamlı seviyeler — good/strong pozitif olduğundan tavan
// olarak elenir. Tek kaynak (form buradan türetir); RiskLevel'a yeni riskli tier eklenirse burası.
export const MAX_RISK_LEVELS: RiskLevel[] = ["medium", "high", "critical"];

// Alarm önem seviyesi Türkçe etiketleri (severityMeta'da label yok — filtre/eşik UI'ı için).
export const ALERT_SEVERITY_LABELS: Record<AlertSeverity, string> = {
  info: "Bilgi",
  positive: "Olumlu",
  warning: "Uyarı",
  critical: "Kritik",
};

// OCP: teslimat kanalı registry'si (Web / Slack / Email / Webhook).
export const DELIVERY_CHANNEL_DEFS: Record<DeliveryChannel, { label: string; icon: LucideIcon }> = {
  web: { label: "Web", icon: Globe },
  slack: { label: "Slack", icon: Hash },
  email: { label: "E-posta", icon: Mail },
  webhook: { label: "Webhook", icon: Webhook },
};

// AlertRuleDraft, kalıcı olmayan form taslağıdır (id/enabled AlertRule'da; form üretmez).
export type AlertRuleDraft = {
  name: string;
  trigger: AlertTriggerType;
  scope: string;
  minLiquidity: number;
  minCreatorScore: number;
  maxRisk: RiskLevel;
  channels: DeliveryChannel[];
};

// validateAlertRule, saf senkron validasyon: alan-hata çiftleri döndürür (boş = geçerli).
export function validateAlertRule(d: AlertRuleDraft): { field: string; msg: string }[] {
  const errs: { field: string; msg: string }[] = [];
  if (!d.name.trim()) errs.push({ field: "name", msg: "İsim zorunlu" });
  if (d.minLiquidity < 0) errs.push({ field: "minLiquidity", msg: "Likidite negatif olamaz" });
  if (d.minCreatorScore < 0 || d.minCreatorScore > 100) errs.push({ field: "minCreatorScore", msg: "Skor 0-100 arası olmalı" });
  if (d.channels.length === 0) errs.push({ field: "channels", msg: "En az bir kanal seç" });
  return errs;
}
