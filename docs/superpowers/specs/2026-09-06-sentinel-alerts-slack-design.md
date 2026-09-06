# SENTINEL Frontend — Alerts + Slack (Ekran 10) Tasarım Spec'i

**Tarih:** 2026-09-06
**Artım:** Frontend Increment 12 — Alerts + Slack (Ekran 10)
**Durum:** Spec — implementasyon öncesi

## Amaç

Tasarım Ekran 10'u (`docs/design/sentinel-ui-ux-design.md` "Alerts ve Slack") frontend-mock olarak
gerçekleştirmek: iki ekran — **`/alerts`** (alarm kuralları + alarm geçmişi) ve **`/slack`** (bildirim
teslimat yapılandırması). Diğer artımlarla (Research, Backtesting, Terminal…) aynı **mock-seam + SDD**
deseni; gerçek Slack teslimatı ve kural persist'i **Backend Alt-proje 3**'e bağlı.

### Yön değişikliği: Telegram → Slack (kullanıcı kararı 2026-08-28)

Bildirim kanalı **Telegram yerine Slack**. Bot + chat ID modeli yerine **Slack webhook/app + channel**
modeli. Mevcut `/telegram` rotası + nav öğesi + Sidebar StatusRow **Slack'e** göre revize edilir.

## Kapsam

**Dahil:**
- `/alerts` ekranı: **Alarm Kuralları** (mock listeden okunan kurallar; kart/tablo + aç/kapa toggle +
  "Yeni Kural" oluşturma formu — kontrollü form + saf validasyon → **simüle** toast) + **Alarm Geçmişi**
  (mevcut `AlertEvent` seam'i, timeline/tablo + severity filtresi).
- `/slack` ekranı (rename `/telegram`): bağlantı durumu (webhook), channel, test bildirimi (simüle),
  önem eşiği, sessiz saatler, alarm şablonları, **Slack mesaj önizleme** bileşeni.
- Shell/nav pivotu: `nav.ts` "Telegram"→"Slack" + path `/telegram`→`/slack`; Sidebar StatusRow
  "Telegram: Bağlı"→"Slack: Bağlı".
- Seam genişlemesi (mock + httpApi→`notReady` + DIP): `AlertRule`, `NotificationConfig` tipleri +
  `getAlertRules()`, `getNotificationConfig()` metotları; qk + hook'lar.

**Kapsam dışı (bilinçli, sessiz düşürme yok):**
- **Gerçek Slack teslimatı / webhook POST** → Backend Alt-proje 3. Bu ekran yalnız yapılandırma UI'ı +
  önizleme; "Test bildirimi gönder" simüle toast.
- **Gerçek kural persist / mutation** → `SentinelApi`'de mutation metodu YOK (yapısal güvenlik, Terminal
  artımı deseni). "Yeni Kural" formu kontrollü + validasyonlu ama submit **simüle** (toast), kalıcı değil.
  Kural listesi mock read.
- Email/Webhook kanallarının kendi config ekranları (yalnız delivery-channel seçeneği olarak listelenir).
- Diğer ekranlardaki "Telegram alert" butonları (Token/Creator header) — bu artımın kapsamı değil,
  ayrı bir dokunuşta Slack'e çevrilecek (tasarım dokümanında işaretli kaldı).

## Mimari

**Global ilkeler (kullanıcı önceliği):** SRP küçük dosyalar, OCP registry'ler (config-driven), DIP
(bileşenler `getApi()`/hook üzerinden, mock import etmez), saf/test edilebilir fonksiyonlar.

### Seam (yeni tipler + metotlar)

`lib/api/types.ts`:
```ts
// Alarm tetikleyici türü (OCP registry ile etiketlenir)
export type AlertTriggerType =
  | "new_mint" | "liquidity_added" | "liquidity_removed" | "creator_sale"
  | "whale_activity" | "holder_growth" | "score_change" | "strategy_signal";

export type DeliveryChannel = "web" | "slack" | "email" | "webhook";

export interface AlertRule {
  id: string;
  name: string;
  trigger: AlertTriggerType;
  scope: string;              // "Tüm tokenlar" | belirli token/creator (serbest metin, mock)
  minLiquidity: number;       // USD; 0 = yok
  minCreatorScore: number;    // 0-100; 0 = yok
  maxRisk: import("@/lib/format").RiskLevel;
  channels: DeliveryChannel[];
  enabled: boolean;
}

export type SlackConnectionState = "connected" | "disconnected" | "error";

export interface NotificationConfig {
  slackState: SlackConnectionState;
  channel: string;            // "#alerts"
  workspace: string;          // görünen ad, mock
  minSeverity: import("@/lib/format").AlertSeverity;  // eşik
  quietHours: { start: string; end: string; enabled: boolean }; // "22:00"/"08:00"
  templates: { trigger: AlertTriggerType; template: string }[]; // mesaj şablonları
  tradeApproval: boolean;     // trade onayı Slack'ten istensin mi
}
```

`lib/api/contract.ts` — `SentinelApi`'ye ekle (READ-ONLY, mutation yok):
```ts
getAlertRules(): Promise<AlertRule[]>;
getNotificationConfig(): Promise<NotificationConfig>;
```
(`getAlerts()` ZATEN var — alarm geçmişi için kullanılır; `subscribeAlerts` de var.)

`lib/api/mock.ts` — temsili `AlertRule[]` + `NotificationConfig` (Slack state=connected,
channel="#alerts", ~3-5 kural, birkaç şablon). `lib/api/http.ts` — iki metot `getJson`. Şu an
`LIVE_ENDPOINTS`'e **eklenmez** (backend yok; mock kalır). `get-query-client.ts` — `qk.alertRules`,
`qk.notificationConfig`. `lib/hooks/queries.ts` — `useAlertRules`, `useNotificationConfig`
(+ mevcut `useAlerts`).

### OCP registry'ler

`lib/alerts/alert-defs.ts`:
- `ALERT_TRIGGER_DEFS: Record<AlertTriggerType, { label; icon; description }>` — form select + kural
  kartı etiketi + Slack önizleme ikonu buradan türetilir.
- `DELIVERY_CHANNEL_DEFS: Record<DeliveryChannel, { label; icon }>`.
- Saf `validateAlertRule(draft): { field: string; msg: string }[]` (isim zorunlu, minLiquidity≥0,
  minCreatorScore 0-100, en az bir kanal).

### Bileşen ağacı (SRP)

**`/alerts` (app route):**
- RSC `page.tsx` → `qk.alertRules` + `qk.alerts` prefetch + HydrationBoundary → `AlertsContent` (client).
- `AlertsContent` — üstte sekme/iki-bölme: "Kurallar" | "Geçmiş".
- `AlertRulesPanel` — `useAlertRules`; `AlertRuleCard` listesi (isim, trigger etiketi, scope, kanallar
  rozetleri, aç/kapa `Switch`) + "Yeni Kural" butonu → `AlertRuleForm` (Sheet/dialog).
- `AlertRuleForm` — kontrollü form (isim, trigger select, scope input, minLiquidity, minCreatorScore,
  maxRisk select, kanal checkbox'ları) + `validateAlertRule` hata span'leri + submit **simüle toast**
  ("Kural kaydedildi (simüle)"), kalıcı değil.
- `AlertHistoryPanel` — `useAlerts`; severity filtresi (`severityMeta`) + `AlertHistoryRow` timeline
  (severity dot + type + token + detail + time). Live: `subscribeAlerts` mevcut `useLiveAlerts` deseni.

**`/slack` (app route, rename):**
- `page.tsx` → `qk.notificationConfig` prefetch → `SlackContent` (client).
- `SlackContent` — `useNotificationConfig`; sol config paneli + sağ **`SlackMessagePreview`**.
- `SlackConnectionCard` — state rozeti (connected/disconnected/error), workspace + channel, "Test
  bildirimi gönder" (simüle toast).
- `NotificationSettings` — önem eşiği select, sessiz saatler (start/end + toggle), trade onayı toggle
  (kontrollü; değişiklikler local state + "değişiklikler simüle" notu — persist yok).
- `AlertTemplateList` — `templates` map → trigger etiketi + şablon metni.
- `SlackMessagePreview` — Slack-block stili mesaj kartı (workspace/channel başlığı + örnek alarm
  render'ı, `ALERT_TRIGGER_DEFS` ikon/etiket + severity renk).

### Shell/nav pivotu

- `components/shell/nav.ts`: `{ label: "Telegram", path: "/telegram", icon: Send }` →
  `{ label: "Slack", path: "/slack", icon: Hash }` (lucide `Hash` — Slack channel çağrışımı; `Send`
  Telegram'a özgüydü).
- `app/(app)/telegram/` → `app/(app)/slack/` (route rename; `git mv`).
- `components/shell/Sidebar.tsx:47` StatusRow "Telegram"/"Bağlı" → "Slack"/"Bağlı" (icon `Hash`).

## Veri akışı

Diğer ekranlarla aynı hibrit seam: `getApi()` → mock (default) veya httpApi (`NEXT_PUBLIC_DATA_SOURCE`).
`getAlertRules`/`getNotificationConfig` `LIVE_ENDPOINTS`'te DEĞİL → her modda mock (backend Alt-proje 3
gelene kadar). `getAlerts` mevcut davranışı korur. React Query poll/prefetch + Zustand yok (bu ekranlar
oturum-durumsuz; form local `useState`).

## Hata yönetimi

- Query loading/error → Skeleton + "Alınamadı" mesajı (mevcut ekran deseni).
- `AlertRuleForm` validasyon → alan altı Türkçe hata span'i + submit gating (Backtesting/Order deseni).
- Boş kural listesi → "Henüz kural yok, ilk kuralını oluştur" boş durumu.
- Slack state=disconnected/error → uyarı rozeti + "bağla" CTA (simüle).

## Test

Vitest + RTL, mevcut desen:
- `mock.test.ts` — `getAlertRules`/`getNotificationConfig` şekil testi.
- `validateAlertRule` saf birim testleri (her hata dalı + geçerli).
- `AlertRuleForm` — validasyon hata render + geçerli submit toast (mock toast).
- `AlertRulesPanel`/`AlertHistoryPanel` — render + toggle + severity filtre.
- `SlackContent`/`SlackMessagePreview` — config render + preview.
- Hiçbir bileşen `mock.ts` import etmez (DIP; getApi/hook üzerinden).
- `npm run build` — `/alerts` + `/slack` prerender; `/telegram` artık yok (404 beklenir).

## Backend'e ertelenen (followups)

- Gerçek Slack app/webhook teslimatı + kural persist/CRUD (mutation seam) → Backend Alt-proje 3.
- `LIVE_ENDPOINTS`'e `getAlertRules`/`getNotificationConfig` ekleme → backend endpoint gelince.
- Token/Creator header'daki "Telegram alert" butonlarının Slack'e çevrilmesi (bu artım dışı).
