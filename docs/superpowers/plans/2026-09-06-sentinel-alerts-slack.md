# Alerts + Slack (Ekran 10) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ekran 10'u frontend-mock gerçekleştir — `/alerts` (alarm kuralları + geçmiş) ve `/slack` (bildirim teslimat config) ekranları + Telegram→Slack shell/nav pivotu.

**Architecture:** Mevcut hibrit seam (`getApi()` mock/http) + React Query + SRP bileşen ağacı + OCP registry'ler. Yeni READ-ONLY seam metotları mock kalır (`LIVE_ENDPOINTS`'te değil; backend Alt-proje 3). Kural oluşturma + test bildirimi **simüle** (mutation yok — yapısal güvenlik). Route `/telegram`→`/slack` rename.

**Tech Stack:** Next.js 16 App Router, TypeScript, TanStack Query, Tailwind v4, shadcn/Base-UI primitives, Vitest + RTL, lucide-react.

**Spec:** `docs/superpowers/specs/2026-09-06-sentinel-alerts-slack-design.md`

## Global Constraints

- **Clean code & SOLID (kullanıcı önceliği):** SRP küçük dosyalar, OCP registry (config-driven), DIP (bileşenler `getApi()`/hook üzerinden — `mock.ts` import ETMEZ), saf/test edilebilir fonksiyonlar.
- **UI dili Türkçe** (mevcut arayüz + mock etiketleri).
- **Mutation yok:** `SentinelApi` yalnız read; kural oluşturma/test bildirimi simüle toast, kalıcı değil.
- **Seam mock kalır:** `getAlertRules`/`getNotificationConfig` `LIVE_ENDPOINTS`'e EKLENMEZ (backend Alt-proje 3).
- Testler `npx vitest run` yeşil; `npx tsc --noEmit` temiz; `npm run build` başarılı.
- Commit sonu satırı: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`

---

## File Structure

**Yeni:**
- `apps/web/lib/alerts/alert-defs.ts` — OCP registry'ler + saf `validateAlertRule`.
- `apps/web/lib/alerts/alert-defs.test.ts`.
- `apps/web/app/(app)/alerts/AlertsContent.tsx` + alt bileşenler (`AlertRulesPanel`, `AlertRuleCard`, `AlertRuleForm`, `AlertHistoryPanel`, `AlertHistoryRow`) — `components/alerts/` altında.
- `apps/web/components/alerts/*.tsx` + testleri.
- `apps/web/app/(app)/slack/page.tsx` + `SlackContent.tsx` + `components/slack/*` (`SlackConnectionCard`, `NotificationSettings`, `AlertTemplateList`, `SlackMessagePreview`).

**Değişecek:**
- `apps/web/lib/api/types.ts` — `AlertTriggerType`, `DeliveryChannel`, `AlertRule`, `SlackConnectionState`, `NotificationConfig`.
- `apps/web/lib/api/contract.ts` — `getAlertRules`, `getNotificationConfig`.
- `apps/web/lib/api/mock.ts` — mock veriler + metotlar.
- `apps/web/lib/api/http.ts` — iki metot.
- `apps/web/lib/get-query-client.ts` — `qk.alertRules`, `qk.notificationConfig`.
- `apps/web/lib/hooks/queries.ts` — `useAlertRules`, `useNotificationConfig`.
- `apps/web/lib/api/mock.test.ts` — şekil testleri.
- `apps/web/components/shell/nav.ts` — Telegram→Slack.
- `apps/web/components/shell/Sidebar.tsx` — StatusRow Telegram→Slack.
- `apps/web/app/(app)/telegram/` → `apps/web/app/(app)/slack/` (git mv).
- `apps/web/app/(app)/alerts/page.tsx` — PlaceholderScreen → RSC prefetch + AlertsContent.

---

## Task 1: Seam — tipler + contract + mock + http + qk + hook'lar

**Files:**
- Modify: `apps/web/lib/api/types.ts`, `contract.ts`, `mock.ts`, `http.ts`, `get-query-client.ts`, `lib/hooks/queries.ts`
- Test: `apps/web/lib/api/mock.test.ts`

**Interfaces:**
- Produces: `AlertTriggerType`, `DeliveryChannel`, `AlertRule`, `SlackConnectionState`, `NotificationConfig` (types); `SentinelApi.getAlertRules()`, `getNotificationConfig()`; `mockApi` impl; `httpApi` impl; `qk.alertRules`, `qk.notificationConfig`; `useAlertRules()`, `useNotificationConfig()`.

- [ ] **Step 1: Tipleri ekle (`types.ts`)** — spec §"Seam" bloğundaki 5 tip aynen.

- [ ] **Step 2: contract'a ekle** — `SentinelApi` interface'ine:
```ts
getAlertRules(): Promise<AlertRule[]>;
getNotificationConfig(): Promise<NotificationConfig>;
```
`import type { ... AlertRule, NotificationConfig }` satırına ekle.

- [ ] **Step 3: Failing test (`mock.test.ts`)**
```ts
it("getAlertRules kurallar döndürür", async () => {
  const rules = await mockApi.getAlertRules();
  expect(Array.isArray(rules)).toBe(true);
  expect(rules.length).toBeGreaterThan(0);
  expect(rules[0]).toHaveProperty("trigger");
  expect(Array.isArray(rules[0].channels)).toBe(true);
});
it("getNotificationConfig Slack config döndürür", async () => {
  const c = await mockApi.getNotificationConfig();
  expect(c.channel).toMatch(/^#/);
  expect(["connected", "disconnected", "error"]).toContain(c.slackState);
  expect(c).toHaveProperty("quietHours");
});
```

- [ ] **Step 4: Run → fail** — `cd apps/web && npx vitest run lib/api/mock.test.ts` → FAIL (metot yok).

- [ ] **Step 5: mock impl (`mock.ts`)** — `import type { AlertRule, NotificationConfig }` ekle. Modül düzeyinde:
```ts
const alertRules: AlertRule[] = [
  { id: "r1", name: "Balina alımı — tüm tokenlar", trigger: "whale_activity", scope: "Tüm tokenlar", minLiquidity: 50000, minCreatorScore: 0, maxRisk: "high", channels: ["web", "slack"], enabled: true },
  { id: "r2", name: "Likidite çekilişi — kritik", trigger: "liquidity_removed", scope: "Tüm tokenlar", minLiquidity: 0, minCreatorScore: 0, maxRisk: "critical", channels: ["web", "slack", "email"], enabled: true },
  { id: "r3", name: "Yüksek skorlu yeni mint", trigger: "new_mint", scope: "Pump.fun", minLiquidity: 10000, minCreatorScore: 70, maxRisk: "medium", channels: ["slack"], enabled: false },
  { id: "r4", name: "Üretici satışı uyarısı", trigger: "creator_sale", scope: "Tüm tokenlar", minLiquidity: 0, minCreatorScore: 0, maxRisk: "high", channels: ["web"], enabled: true },
];
const notificationConfig: NotificationConfig = {
  slackState: "connected", channel: "#alerts", workspace: "Sentinel HQ",
  minSeverity: "warning",
  quietHours: { start: "23:00", end: "07:00", enabled: false },
  templates: [
    { trigger: "liquidity_removed", template: "🚨 {{token}}: likidite çekildi — {{detail}}" },
    { trigger: "whale_activity", template: "🐋 {{token}}: balina hareketi — {{detail}}" },
    { trigger: "new_mint", template: "✨ Yeni mint: {{token}} — {{detail}}" },
  ],
  tradeApproval: true,
};
```
`mockApi` nesnesine: `getAlertRules: () => delay(alertRules),` ve `getNotificationConfig: () => delay(notificationConfig),`.

- [ ] **Step 6: http impl (`http.ts`)** — `import type { AlertRule, NotificationConfig }`; `getAlertRules: () => getJson<AlertRule[]>("/api/alert-rules"), getNotificationConfig: () => getJson<NotificationConfig>("/api/notification-config"),`. (`LIVE_ENDPOINTS`'e EKLEME.)

- [ ] **Step 7: qk + hook** — `get-query-client.ts` `qk`'ye: `alertRules: ["alert-rules"] as const, notificationConfig: ["notification-config"] as const,`. `queries.ts`'e:
```ts
export function useAlertRules() {
  return useQuery({ queryKey: qk.alertRules, queryFn: () => getApi().getAlertRules() });
}
export function useNotificationConfig() {
  return useQuery({ queryKey: qk.notificationConfig, queryFn: () => getApi().getNotificationConfig() });
}
```

- [ ] **Step 8: Run → pass + typecheck** — `npx vitest run lib/api/mock.test.ts && npx tsc --noEmit` → PASS.

- [ ] **Step 9: Commit** — `git add apps/web/lib && git commit -m "feat(alerts): AlertRule + NotificationConfig seam (mock read-only) (Task 1)"`

---

## Task 2: OCP registry'ler + saf `validateAlertRule`

**Files:**
- Create: `apps/web/lib/alerts/alert-defs.ts`, `apps/web/lib/alerts/alert-defs.test.ts`

**Interfaces:**
- Consumes: `AlertTriggerType`, `DeliveryChannel`, `AlertRule`, `RiskLevel`.
- Produces: `ALERT_TRIGGER_DEFS`, `DELIVERY_CHANNEL_DEFS`, `validateAlertRule(draft): {field,msg}[]`, `AlertRuleDraft` type.

- [ ] **Step 1: Failing test (`alert-defs.test.ts`)**
```ts
import { describe, it, expect } from "vitest";
import { validateAlertRule, ALERT_TRIGGER_DEFS, DELIVERY_CHANNEL_DEFS } from "./alert-defs";

const valid = { name: "X", trigger: "new_mint", scope: "Tüm tokenlar", minLiquidity: 0, minCreatorScore: 50, maxRisk: "medium", channels: ["slack"] } as const;

describe("validateAlertRule", () => {
  it("geçerli taslak → hata yok", () => {
    expect(validateAlertRule(valid)).toEqual([]);
  });
  it("boş isim → name hatası", () => {
    expect(validateAlertRule({ ...valid, name: " " }).some(e => e.field === "name")).toBe(true);
  });
  it("negatif likidite → minLiquidity hatası", () => {
    expect(validateAlertRule({ ...valid, minLiquidity: -1 }).some(e => e.field === "minLiquidity")).toBe(true);
  });
  it("skor 0-100 dışı → minCreatorScore hatası", () => {
    expect(validateAlertRule({ ...valid, minCreatorScore: 150 }).some(e => e.field === "minCreatorScore")).toBe(true);
  });
  it("kanal yok → channels hatası", () => {
    expect(validateAlertRule({ ...valid, channels: [] }).some(e => e.field === "channels")).toBe(true);
  });
});
describe("registries", () => {
  it("her trigger etiketli", () => {
    expect(ALERT_TRIGGER_DEFS.new_mint.label).toBeTruthy();
    expect(Object.keys(DELIVERY_CHANNEL_DEFS)).toContain("slack");
  });
});
```

- [ ] **Step 2: Run → fail** — `npx vitest run lib/alerts/alert-defs.test.ts` → FAIL.

- [ ] **Step 3: Implement `alert-defs.ts`**
```ts
import type { AlertTriggerType, DeliveryChannel, RiskLevel } from "@/lib/api/types";
import { Sparkles, Droplet, DropletOff, TrendingDown, Fish, Users, Activity, Zap, Globe, Hash, Mail, Webhook, type LucideIcon } from "lucide-react";

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

export const DELIVERY_CHANNEL_DEFS: Record<DeliveryChannel, { label: string; icon: LucideIcon }> = {
  web: { label: "Web", icon: Globe },
  slack: { label: "Slack", icon: Hash },
  email: { label: "E-posta", icon: Mail },
  webhook: { label: "Webhook", icon: Webhook },
};

export type AlertRuleDraft = {
  name: string; trigger: AlertTriggerType; scope: string;
  minLiquidity: number; minCreatorScore: number; maxRisk: RiskLevel; channels: DeliveryChannel[];
};

export function validateAlertRule(d: AlertRuleDraft): { field: string; msg: string }[] {
  const errs: { field: string; msg: string }[] = [];
  if (!d.name.trim()) errs.push({ field: "name", msg: "İsim zorunlu" });
  if (d.minLiquidity < 0) errs.push({ field: "minLiquidity", msg: "Likidite negatif olamaz" });
  if (d.minCreatorScore < 0 || d.minCreatorScore > 100) errs.push({ field: "minCreatorScore", msg: "Skor 0-100 arası olmalı" });
  if (d.channels.length === 0) errs.push({ field: "channels", msg: "En az bir kanal seç" });
  return errs;
}
```

- [ ] **Step 4: Run → pass** — `npx vitest run lib/alerts/alert-defs.test.ts` → PASS.

- [ ] **Step 5: Commit** — `git add apps/web/lib/alerts && git commit -m "feat(alerts): ALERT_TRIGGER/DELIVERY_CHANNEL registries + validateAlertRule (Task 2)"`

---

## Task 3: Shell/nav pivotu — Telegram→Slack + route rename

**Files:**
- Modify: `apps/web/components/shell/nav.ts`, `apps/web/components/shell/Sidebar.tsx`
- Rename: `apps/web/app/(app)/telegram/page.tsx` → `apps/web/app/(app)/slack/page.tsx` (git mv)
- Test: `apps/web/components/shell/nav.test.ts` (varsa; yoksa build ile doğrula)

**Interfaces:**
- Produces: nav item `{ label: "Slack", path: "/slack", icon: Hash }`; Sidebar StatusRow "Slack".

- [ ] **Step 1: nav.ts** — `Send` import'unu `Hash` ile değiştir (kullanılan tek yer burasıysa); satırı:
```ts
{ label: "Slack", path: "/slack", icon: Hash },
```

- [ ] **Step 2: Route rename** — `git mv "apps/web/app/(app)/telegram" "apps/web/app/(app)/slack"` (Task 8'de içerik değişecek; şimdilik PlaceholderScreen taşınır).

- [ ] **Step 3: Sidebar StatusRow** — `Sidebar.tsx:47` `<StatusRow icon={<Send size={12} />} label="Telegram" value="Bağlı" ok />` → `label="Slack"` + icon `Hash` (import düzelt; `Send` başka yerde kullanılmıyorsa kaldır).

- [ ] **Step 4: Run → build + typecheck** — `npx tsc --noEmit && npm run build` → başarılı; `/slack` prerender, `/telegram` yok.

- [ ] **Step 5: Commit** — `git add apps/web/components/shell apps/web/app && git commit -m "feat(alerts): shell/nav Telegram→Slack pivotu + /telegram→/slack rename (Task 3)"`

---

## Task 4: `/alerts` — AlertHistoryPanel (severity filtre + timeline)

**Files:**
- Create: `apps/web/components/alerts/AlertHistoryPanel.tsx`, `AlertHistoryRow.tsx`
- Test: `apps/web/components/alerts/AlertHistoryPanel.test.tsx`

**Interfaces:**
- Consumes: `useAlerts` (mevcut), `AlertEvent`, `severityMeta`, `AlertSeverity`.
- Produces: `AlertHistoryPanel` (default export component).

- [ ] **Step 1: Failing test** (mock `useAlerts` — `vi.mock("@/lib/hooks/queries")`)
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertHistoryPanel from "./AlertHistoryPanel";
vi.mock("@/lib/hooks/queries", () => ({
  useAlerts: () => ({ data: [
    { id: "a1", type: "Balina Alımı", token: "PULSE", detail: "x", severity: "positive", time: "az önce" },
    { id: "a2", type: "Likidite Çekildi", token: "GFROG", detail: "y", severity: "critical", time: "18sn önce" },
  ], isLoading: false, isError: false }),
}));
describe("AlertHistoryPanel", () => {
  it("alarmları listeler", () => {
    render(<AlertHistoryPanel />);
    expect(screen.getByText("PULSE")).toBeInTheDocument();
    expect(screen.getByText("GFROG")).toBeInTheDocument();
  });
  it("severity filtresi daraltır", () => {
    render(<AlertHistoryPanel />);
    fireEvent.click(screen.getByRole("button", { name: /kritik/i }));
    expect(screen.queryByText("PULSE")).not.toBeInTheDocument();
    expect(screen.getByText("GFROG")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run → fail** — `npx vitest run components/alerts/AlertHistoryPanel.test.tsx`.

- [ ] **Step 3: Implement** — `AlertHistoryRow.tsx`: bir `AlertEvent` alır; severity dot (`severityMeta[severity].dot`) + type + token (font-mono) + detail + time. `AlertHistoryPanel.tsx`: `"use client"`; `useAlerts()`; local `useState<AlertSeverity | "all">("all")`; filtre çipleri (`severityMeta` anahtarları + "Tümü") `aria-pressed`; filtrelenmiş listeyi `AlertHistoryRow` ile render; loading→Skeleton, error→mesaj, boş→"Alarm yok".

- [ ] **Step 4: Run → pass** — PASS.

- [ ] **Step 5: Commit** — `git add apps/web/components/alerts && git commit -m "feat(alerts): AlertHistoryPanel + severity filtre + timeline (Task 4)"`

---

## Task 5: `/alerts` — AlertRulesPanel + AlertRuleCard (+ toggle)

**Files:**
- Create: `apps/web/components/alerts/AlertRulesPanel.tsx`, `AlertRuleCard.tsx`
- Test: `apps/web/components/alerts/AlertRulesPanel.test.tsx`

**Interfaces:**
- Consumes: `useAlertRules`, `AlertRule`, `ALERT_TRIGGER_DEFS`, `DELIVERY_CHANNEL_DEFS`.
- Produces: `AlertRulesPanel` (default; prop `onNew?: () => void`), `AlertRuleCard` (prop `rule`, `onToggle`).

- [ ] **Step 1: Failing test** (mock `useAlertRules`)
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertRulesPanel from "./AlertRulesPanel";
vi.mock("@/lib/hooks/queries", () => ({
  useAlertRules: () => ({ data: [
    { id: "r1", name: "Kural Bir", trigger: "whale_activity", scope: "Tüm tokenlar", minLiquidity: 0, minCreatorScore: 0, maxRisk: "high", channels: ["slack"], enabled: true },
  ], isLoading: false, isError: false }),
}));
describe("AlertRulesPanel", () => {
  it("kuralları + trigger etiketini gösterir", () => {
    render(<AlertRulesPanel />);
    expect(screen.getByText("Kural Bir")).toBeInTheDocument();
    expect(screen.getByText("Balina Hareketi")).toBeInTheDocument();
  });
  it("toggle local state'i çevirir", () => {
    render(<AlertRulesPanel />);
    const sw = screen.getByRole("switch");
    expect(sw).toBeChecked();
    fireEvent.click(sw);
    expect(sw).not.toBeChecked();
  });
});
```

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — `AlertRuleCard.tsx`: `rule` + `onToggle(id)`; isim, `ALERT_TRIGGER_DEFS[trigger]` etiket+ikon, scope, kanal rozetleri (`DELIVERY_CHANNEL_DEFS`), `<Switch role="switch" checked={rule.enabled}>` (mevcut shadcn Switch varsa reuse; yoksa `<button role="switch" aria-checked>`). `AlertRulesPanel.tsx`: `"use client"`; `useAlertRules()`; local `useState` ile enabled override map (toggle simüle — persist yok, yorum düş); "Yeni Kural" butonu `onNew`; loading/error/boş durumlar.

- [ ] **Step 4: Run → pass.**

- [ ] **Step 5: Commit** — `git add apps/web/components/alerts && git commit -m "feat(alerts): AlertRulesPanel + AlertRuleCard + simüle toggle (Task 5)"`

---

## Task 6: `/alerts` — AlertRuleForm (kontrollü + validate + simüle submit)

**Files:**
- Create: `apps/web/components/alerts/AlertRuleForm.tsx`
- Test: `apps/web/components/alerts/AlertRuleForm.test.tsx`

**Interfaces:**
- Consumes: `validateAlertRule`, `AlertRuleDraft`, `ALERT_TRIGGER_DEFS`, `DELIVERY_CHANNEL_DEFS`, `RiskLevel`, `sonner` toast.
- Produces: `AlertRuleForm` (props `onDone?: () => void`).

- [ ] **Step 1: Failing test** (`vi.mock("sonner", ...)`)
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertRuleForm from "./AlertRuleForm";
const toast = vi.fn();
vi.mock("sonner", () => ({ toast: (...a: unknown[]) => toast(...a) }));
describe("AlertRuleForm", () => {
  it("boş isim → hata gösterir, submit engellenir", () => {
    render(<AlertRuleForm />);
    fireEvent.click(screen.getByRole("button", { name: /kaydet/i }));
    expect(screen.getByText(/isim zorunlu/i)).toBeInTheDocument();
  });
  it("geçerli form → simüle toast", () => {
    render(<AlertRuleForm />);
    fireEvent.change(screen.getByLabelText(/isim/i), { target: { value: "Yeni kural" } });
    fireEvent.click(screen.getByRole("button", { name: /kaydet/i }));
    expect(toast).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — `"use client"`; kontrollü `useState<AlertRuleDraft>` (default trigger `new_mint`, maxRisk `medium`, channels `["slack"]`, scope "Tüm tokenlar", sayısal alanlar 0); alanlar: isim `<input aria-label>`, trigger `<select>` (`ALERT_TRIGGER_DEFS`), scope input, minLiquidity/minCreatorScore number, maxRisk select (`RISK` seviyeleri — `lib/format` riskMeta anahtarları), kanal checkbox'ları (`DELIVERY_CHANNEL_DEFS`); submit → `validateAlertRule`; hata varsa alan altı span; yoksa `toast.success("Kural kaydedildi (simüle — kalıcı değil)")` + `onDone?.()`. Yorum: gerçek persist Backend Alt-proje 3.

- [ ] **Step 4: Run → pass.**

- [ ] **Step 5: Commit** — `git add apps/web/components/alerts && git commit -m "feat(alerts): AlertRuleForm kontrollü + validate + simüle submit (Task 6)"`

---

## Task 7: `/alerts` — AlertsContent (sekmeler) + page RSC

**Files:**
- Create: `apps/web/app/(app)/alerts/AlertsContent.tsx`
- Modify: `apps/web/app/(app)/alerts/page.tsx`
- Test: `apps/web/app/(app)/alerts/AlertsContent.test.tsx`

**Interfaces:**
- Consumes: `AlertRulesPanel`, `AlertHistoryPanel`, `AlertRuleForm`.
- Produces: `AlertsContent` (default), `page` (RSC).

- [ ] **Step 1: Failing test** (paneller mock'lanır)
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import AlertsContent from "./AlertsContent";
vi.mock("@/components/alerts/AlertRulesPanel", () => ({ default: ({ onNew }: { onNew: () => void }) => <button onClick={onNew}>rules-panel</button> }));
vi.mock("@/components/alerts/AlertHistoryPanel", () => ({ default: () => <div>history-panel</div> }));
vi.mock("@/components/alerts/AlertRuleForm", () => ({ default: () => <div>rule-form</div> }));
describe("AlertsContent", () => {
  it("sekmeler arası geçiş", () => {
    render(<AlertsContent />);
    expect(screen.getByText("rules-panel")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: /geçmiş/i }));
    expect(screen.getByText("history-panel")).toBeInTheDocument();
  });
  it("Yeni Kural formu açar", () => {
    render(<AlertsContent />);
    fireEvent.click(screen.getByText("rules-panel"));
    expect(screen.getByText("rule-form")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — `AlertsContent.tsx`: `"use client"`; başlık "Uyarılar"; iki sekme (`role="tab"`) "Kurallar"|"Geçmiş" (local state); Kurallar → `AlertRulesPanel onNew={()=>setFormOpen(true)}` + `AlertRuleForm` (Sheet/inline, `formOpen`); Geçmiş → `AlertHistoryPanel`. `page.tsx`: RSC — `getQueryClient()` + `prefetchQuery(qk.alertRules)` + `prefetchQuery(qk.alerts)` + `HydrationBoundary` → `<AlertsContent/>` (mevcut ekranların RSC deseni; ör. backtesting/system-health page'ine bak).

- [ ] **Step 4: Run → pass + build** — `npx vitest run app/(app)/alerts && npm run build`.

- [ ] **Step 5: Commit** — `git add "apps/web/app/(app)/alerts" && git commit -m "feat(alerts): AlertsContent sekmeler + /alerts RSC prefetch (Task 7)"`

---

## Task 8: `/slack` — SlackContent + config bileşenleri

**Files:**
- Create: `apps/web/app/(app)/slack/SlackContent.tsx`, `apps/web/components/slack/SlackConnectionCard.tsx`, `NotificationSettings.tsx`, `AlertTemplateList.tsx`
- Modify: `apps/web/app/(app)/slack/page.tsx`
- Test: `apps/web/app/(app)/slack/SlackContent.test.tsx`

**Interfaces:**
- Consumes: `useNotificationConfig`, `NotificationConfig`, `ALERT_TRIGGER_DEFS`, `severityMeta`, `sonner` toast.
- Produces: `SlackContent` (default), `SlackConnectionCard`, `NotificationSettings`, `AlertTemplateList`, `page` (RSC).

- [ ] **Step 1: Failing test** (mock `useNotificationConfig` + `sonner`)
```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import SlackContent from "./SlackContent";
const toast = vi.fn();
vi.mock("sonner", () => ({ toast: Object.assign((...a: unknown[]) => toast(...a), { success: (...a: unknown[]) => toast(...a) }) }));
vi.mock("@/lib/hooks/queries", () => ({
  useNotificationConfig: () => ({ data: {
    slackState: "connected", channel: "#alerts", workspace: "Sentinel HQ", minSeverity: "warning",
    quietHours: { start: "23:00", end: "07:00", enabled: false },
    templates: [{ trigger: "new_mint", template: "✨ {{token}}" }], tradeApproval: true,
  }, isLoading: false, isError: false }),
}));
describe("SlackContent", () => {
  it("bağlantı + channel gösterir", () => {
    render(<SlackContent />);
    expect(screen.getByText("#alerts")).toBeInTheDocument();
    expect(screen.getByText(/Sentinel HQ/)).toBeInTheDocument();
  });
  it("test bildirimi → simüle toast", () => {
    render(<SlackContent />);
    fireEvent.click(screen.getByRole("button", { name: /test bildirimi/i }));
    expect(toast).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — `SlackConnectionCard.tsx`: state rozeti (connected=yeşil/disconnected=zinc/error=kırmızı), workspace + channel (font-mono), "Test bildirimi gönder" → `toast.success("Test bildirimi gönderildi (simüle)")`. `NotificationSettings.tsx`: önem eşiği `<select>` (`severityMeta` anahtarları), sessiz saatler start/end input + toggle, trade onayı toggle — kontrollü local state, "değişiklikler simüle" notu. `AlertTemplateList.tsx`: `templates.map` → `ALERT_TRIGGER_DEFS[trigger].label` + şablon metni (font-mono). `SlackContent.tsx`: `"use client"`; başlık "Slack"; `useNotificationConfig()`; sol config (üç bileşen) + sağ preview placeholder (Task 9'da doldurulur — şimdilik `<SlackMessagePreview config=...>` yoksa geçici boş div, Task 9 ekler). loading/error. `page.tsx`: RSC prefetch `qk.notificationConfig` + HydrationBoundary.

- [ ] **Step 4: Run → pass + build.**

- [ ] **Step 5: Commit** — `git add "apps/web/app/(app)/slack" apps/web/components/slack && git commit -m "feat(alerts): /slack SlackContent + bağlantı/ayar/şablon bileşenleri (Task 8)"`

---

## Task 9: `/slack` — SlackMessagePreview

**Files:**
- Create: `apps/web/components/slack/SlackMessagePreview.tsx`
- Modify: `apps/web/app/(app)/slack/SlackContent.tsx` (preview'ı bağla)
- Test: `apps/web/components/slack/SlackMessagePreview.test.tsx`

**Interfaces:**
- Consumes: `NotificationConfig`, `ALERT_TRIGGER_DEFS`, `severityMeta`.
- Produces: `SlackMessagePreview` (prop `config: NotificationConfig`).

- [ ] **Step 1: Failing test**
```tsx
import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import SlackMessagePreview from "./SlackMessagePreview";
const config = { slackState: "connected", channel: "#alerts", workspace: "Sentinel HQ", minSeverity: "warning", quietHours: { start: "23:00", end: "07:00", enabled: false }, templates: [{ trigger: "liquidity_removed", template: "🚨 {{token}}: {{detail}}" }], tradeApproval: true } as const;
describe("SlackMessagePreview", () => {
  it("channel + örnek mesaj render eder", () => {
    render(<SlackMessagePreview config={config} />);
    expect(screen.getByText("#alerts")).toBeInTheDocument();
    expect(screen.getByText(/Likidite Çekildi|GFROG|🚨/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run → fail.**

- [ ] **Step 3: Implement** — Slack-block stili kart: üst bar workspace/channel, "Sentinel APP" rozeti, gövde ilk `templates[0]` (yoksa fallback) trigger `ALERT_TRIGGER_DEFS` ikon+etiket + örnek token/detail (`{{token}}`→"GFROG", `{{detail}}`→örnek), severity renk şeridi. `SlackContent.tsx`'te sağ sütuna `<SlackMessagePreview config={data} />` bağla.

- [ ] **Step 4: Run → pass + build.**

- [ ] **Step 5: Commit** — `git add apps/web/components/slack "apps/web/app/(app)/slack" && git commit -m "feat(alerts): SlackMessagePreview + SlackContent'e bağla (Task 9)"`

---

## Task 10: Whole-branch review + doküman + görsel doğrulama

- [ ] **Step 1: Tüm testler + build + typecheck** — `cd apps/web && npx tsc --noEmit && npx vitest run && npm run lint && npm run build` → hepsi yeşil; `/alerts` + `/slack` prerender.
- [ ] **Step 2: Whole-branch review** — superpowers:requesting-code-review (opus). Bulguları superpowers:receiving-code-review ile ele al.
- [ ] **Step 3: Yaşayan dokümanlar** — `docs/progress.md` (Increment 12 girişi) + `MEMORY.md` aktif-iş satırı; `docs/superpowers/followups-frontend.md`'ye ertelenenler (gerçek Slack teslimat + kural persist + LIVE_ENDPOINTS + diğer ekranların Telegram→Slack butonları).
- [ ] **Step 4: Görsel doğrulama** — dev'de `/alerts` (sekmeler, kurallar, form validasyon+simüle toast, geçmiş filtre) + `/slack` (bağlantı, ayarlar, şablonlar, preview).
- [ ] **Step 5: Merge/push** — kullanıcı onayıyla (DUR-noktası).

---

## Self-Review

**1. Spec coverage:**
- Seam (AlertRule/NotificationConfig + metotlar) → Task 1 ✅
- OCP registry + validate → Task 2 ✅
- Shell/nav Telegram→Slack + rename → Task 3 ✅
- `/alerts` geçmiş → Task 4 ✅; kurallar → Task 5; form → Task 6; sekme+RSC → Task 7 ✅
- `/slack` config → Task 8; preview → Task 9 ✅
- Kapsam dışı (gerçek teslimat/persist/LIVE_ENDPOINTS) → Task 1 (mock kalır) + Task 10 followups ✅

**2. Placeholder scan:** Tüm test kodları + impl adımları somut; "mevcut ekran deseni" referansları (RSC prefetch, Switch primitive) task-anında ilgili dosyaya bakılarak uygulanır — iskelet + tip tam.

**3. Type consistency:** `AlertRule`/`NotificationConfig`/`AlertTriggerType`/`DeliveryChannel` Task 1'de tanımlı, Task 2-9'da aynı adlarla tüketilir; `validateAlertRule`/`AlertRuleDraft` Task 2 → Task 6; `ALERT_TRIGGER_DEFS`/`DELIVERY_CHANNEL_DEFS` Task 2 → Task 5/6/8/9; `useAlertRules`/`useNotificationConfig` Task 1 → Task 5/8; `getAlerts`/`useAlerts` mevcut → Task 4. Tutarlı.
