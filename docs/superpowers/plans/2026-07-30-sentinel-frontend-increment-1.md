# SENTINEL Frontend — Increment 1 Implementation Plan
### İskele + Design System + App Shell + Overview Dashboard

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Çalışan bir Next.js (App Router, server-first) uygulaması kur; Sentinel dark design system'i, app shell'i (sidebar/header/trading-mode) ve tam işlevsel Overview Dashboard'u mock veri seam'i üzerinden hayata geçir.

**Architecture:** Monorepo `apps/web/`. Server Components layout + ilk veri prefetch (RSC → TanStack Query dehydrate/hydrate); yalnız etkileşimli/canlı adacıklar `"use client"`. Tüm veri `component → hook → getApi() → (mock|http)` seam'inden akar — hiçbir bileşen mock'u doğrudan import etmez. Backend gelince yalnız `http.ts` yazılır ve env değişir.

**Tech Stack:** Next.js 15 (App Router), React 19, TypeScript (strict), Tailwind CSS v4, shadcn/ui (Base UI-based — `@base-ui/react`, shadcn'in güncel varsayılanı, Radix'in halefi), TanStack Query v5, Zustand, Recharts, lucide-react, sonner, Vitest + React Testing Library.

## Global Constraints
- **Konum:** tüm iş `SENTINEL/apps/web/` altında. Komutlar bu dizinden çalışır.
- **Node:** 20+ · **Paket yöneticisi:** npm.
- **Dark-only:** kök `<html class="dark">`; light tema kapsam dışı.
- **Veri erişimi:** bileşenler `lib/api/mock`'u **doğrudan import edemez**; yalnız `lib/hooks/*` → `getApi()`.
- **Mono font:** yalnız cüzdan/mint adresi, tx hash, teknik metrik (`font-mono`); geri kalan Inter.
- **Skor seviyeleri (tek kaynak `lib/format.ts`):** `0–24 Critical · 25–49 High · 50–69 Medium · 70–84 Good · 85–100 Strong`. Renk + her zaman metin etiketi.
- **Sentinel renk token'ları:** bg `#080B12`, surface-1 `#111722`, surface-2 `#151C28`, surface-3 `#1A2331`, foreground `#E6EAF2`, muted `#8A94A6`, primary `#7C5CFF`, accent-blue/info `#3E9BFF`, positive `#2FD98B`, warning `#FFB020`, critical `#F0476B`, border `rgba(255,255,255,0.07)`, radius `0.625rem`.
- **Commit:** her task sonunda commit. **Prerequisite:** execution başında `git init` gerekir — kullanıcı git init'i henüz istemedi; execution'a başlamadan önce onay al. Onaya kadar commit adımları çalıştırılmaz (kod yazılır, commit ertelenir).
- **Kaynak referans:** ported kod Figma Make `zAWGuUwmbKkSK0n7YKlANt` implementasyonundan uyarlanmıştır (react-router → Next; doğrudan mock import → hook seam).
- shadcn primitive'leri Base UI tabanlıdır (`@base-ui/react`); port edilen referans bileşenler Radix `asChild` yerine `render` prop kullanır.

---

### Task 1: Scaffold Next.js app + Tailwind v4 + tokens + fonts + test harness

**Files:**
- Create: `apps/web/` (Next.js proje ağacı)
- Create: `apps/web/app/layout.tsx`, `apps/web/app/globals.css`, `apps/web/app/page.tsx` (geçici)
- Create: `apps/web/vitest.config.ts`, `apps/web/test/setup.ts`, `apps/web/test/smoke.test.tsx`
- Create: `apps/web/.env.example`
- Modify: `apps/web/tsconfig.json` (strict + path alias `@/*`)

**Interfaces:**
- Produces: `@/*` path alias (kök `apps/web/`), `bg-background/text-foreground/bg-card` gibi Sentinel utility'leri, `font-mono` / Inter varsayılanı, çalışan Vitest+RTL harness.

- [ ] **Step 1: Next.js projesini oluştur**

```bash
cd apps/web 2>/dev/null || mkdir -p apps/web && cd apps/web
# Boş dizine kurulum (interactive olmayan)
npx create-next-app@latest . --ts --app --tailwind --eslint --src-dir=false --import-alias "@/*" --no-turbopack --use-npm --yes
```

Beklenen: `app/`, `package.json`, `tsconfig.json`, Tailwind v4 kurulu.

- [ ] **Step 2: Test bağımlılıkları ve grafik/ikon/veri kütüphaneleri**

```bash
npm i @tanstack/react-query zustand recharts lucide-react sonner clsx tailwind-merge class-variance-authority
npm i -D vitest @vitejs/plugin-react jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event
```

- [ ] **Step 3: `vitest.config.ts` yaz**

```ts
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { "@": fileURLToPath(new URL("./", import.meta.url)) } },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./test/setup.ts"],
    css: false,
  },
});
```

- [ ] **Step 4: `test/setup.ts` yaz**

```ts
import "@testing-library/jest-dom/vitest";
```

- [ ] **Step 5: `app/globals.css` — Sentinel dark token'larını yaz** (referans `theme.css` portu, dark-only)

```css
@import "tailwindcss";

@custom-variant dark (&:is(.dark *));

:root {
  --font-sans: var(--font-inter), ui-sans-serif, system-ui, sans-serif;
  --font-mono: var(--font-jetbrains), "JetBrains Mono", ui-monospace, monospace;
  --font-weight-medium: 500;
  --font-weight-normal: 400;
  --radius: 0.625rem;
}

.dark {
  --background: #080B12;
  --foreground: #E6EAF2;
  --card: #111722;
  --card-foreground: #E6EAF2;
  --popover: #151C28;
  --popover-foreground: #E6EAF2;
  --primary: #7C5CFF;
  --primary-foreground: #FFFFFF;
  --secondary: #1A2331;
  --secondary-foreground: #E6EAF2;
  --muted: #151C28;
  --muted-foreground: #8A94A6;
  --accent: #1A2331;
  --accent-foreground: #E6EAF2;
  --destructive: #F0476B;
  --destructive-foreground: #FFFFFF;
  --border: rgba(255, 255, 255, 0.07);
  --input: #1A2331;
  --input-background: #111722;
  --ring: #7C5CFF;

  --sentinel-accent: #7C5CFF;
  --sentinel-accent-blue: #3E9BFF;
  --sentinel-positive: #2FD98B;
  --sentinel-warning: #FFB020;
  --sentinel-critical: #F0476B;
  --sentinel-info: #3E9BFF;
  --sentinel-neutral: #8A94A6;
  --sentinel-surface-1: #111722;
  --sentinel-surface-2: #151C28;
  --sentinel-surface-3: #1A2331;

  --chart-1: #7C5CFF;
  --chart-2: #2FD98B;
  --chart-3: #3E9BFF;
  --chart-4: #FFB020;
  --chart-5: #F0476B;

  --sidebar: #0B1019;
  --sidebar-foreground: #C4CBD8;
  --sidebar-primary: #7C5CFF;
  --sidebar-primary-foreground: #FFFFFF;
  --sidebar-accent: #1A2331;
  --sidebar-accent-foreground: #FFFFFF;
  --sidebar-border: rgba(255, 255, 255, 0.06);
  --sidebar-ring: #7C5CFF;
}

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-card-foreground: var(--card-foreground);
  --color-popover: var(--popover);
  --color-popover-foreground: var(--popover-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-accent-foreground: var(--accent-foreground);
  --color-destructive: var(--destructive);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-input-background: var(--input-background);
  --color-ring: var(--ring);
  --color-chart-1: var(--chart-1);
  --color-chart-2: var(--chart-2);
  --color-chart-3: var(--chart-3);
  --color-chart-4: var(--chart-4);
  --color-chart-5: var(--chart-5);
  --color-positive: var(--sentinel-positive);
  --color-warning: var(--sentinel-warning);
  --color-critical: var(--sentinel-critical);
  --color-info: var(--sentinel-info);
  --color-neutral: var(--sentinel-neutral);
  --color-accent-blue: var(--sentinel-accent-blue);
  --color-surface-1: var(--sentinel-surface-1);
  --color-surface-2: var(--sentinel-surface-2);
  --color-surface-3: var(--sentinel-surface-3);
  --color-sidebar: var(--sidebar);
  --color-sidebar-foreground: var(--sidebar-foreground);
  --color-sidebar-primary: var(--sidebar-primary);
  --color-sidebar-primary-foreground: var(--sidebar-primary-foreground);
  --color-sidebar-accent: var(--sidebar-accent);
  --color-sidebar-accent-foreground: var(--sidebar-accent-foreground);
  --color-sidebar-border: var(--sidebar-border);
  --radius-sm: calc(var(--radius) - 4px);
  --radius-md: calc(var(--radius) - 2px);
  --radius-lg: var(--radius);
  --font-mono: var(--font-mono);
}

@layer base {
  * { border-color: var(--border); }
  body {
    background-color: var(--background);
    color: var(--foreground);
    font-family: var(--font-sans);
    -webkit-font-smoothing: antialiased;
  }
  .font-mono { font-family: var(--font-mono); }
  h1 { font-size: 1.5rem; font-weight: 500; line-height: 1.4; }
  h2 { font-size: 1.25rem; font-weight: 500; line-height: 1.4; }
  h3 { font-size: 1.0625rem; font-weight: 500; line-height: 1.4; }
}
```

- [ ] **Step 6: `app/layout.tsx` — kök layout (dark, fontlar)**

```tsx
import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });
const jetbrains = JetBrains_Mono({ subsets: ["latin"], variable: "--font-jetbrains" });

export const metadata: Metadata = {
  title: "Sentinel — Solana Token Intelligence",
  description: "Real-time Solana token intelligence and trading terminal.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`dark ${inter.variable} ${jetbrains.variable}`}>
      <body>{children}</body>
    </html>
  );
}
```

- [ ] **Step 7: geçici `app/page.tsx`**

```tsx
export default function Home() {
  return <main className="p-6"><h1>Sentinel</h1></main>;
}
```

- [ ] **Step 8: `.env.example`**

```
NEXT_PUBLIC_DATA_SOURCE=mock
NEXT_PUBLIC_API_BASE_URL=
NEXT_PUBLIC_WS_URL=
```

- [ ] **Step 9: `test/smoke.test.tsx` — harness doğrulaması**

```tsx
import { render, screen } from "@testing-library/react";
import Home from "@/app/page";

test("home renders Sentinel heading", () => {
  render(<Home />);
  expect(screen.getByRole("heading", { name: "Sentinel" })).toBeInTheDocument();
});
```

- [ ] **Step 10: `package.json`'a test script ekle** — `"scripts"` içine: `"test": "vitest run"`, `"test:watch": "vitest"`.

- [ ] **Step 11: Testleri ve build'i doğrula**

Run: `npm run test`
Expected: PASS (smoke test)
Run: `npm run build`
Expected: derleme başarılı, hata yok.

- [ ] **Step 12: Commit**

```bash
git add apps/web
git commit -m "feat(web): scaffold Next.js app with Sentinel dark tokens and vitest harness"
```

---

### Task 2: `lib/format.ts` — saf formatter + skor mantığı (TDD)

**Files:**
- Create: `apps/web/lib/format.ts`
- Test: `apps/web/lib/format.test.ts`

**Interfaces:**
- Produces:
  - `type RiskLevel = "critical" | "high" | "medium" | "good" | "strong"`
  - `type AlertSeverity = "info" | "positive" | "warning" | "critical"`
  - `scoreToLevel(score: number): RiskLevel`
  - `riskMeta: Record<RiskLevel, { label: string; color: string; bg: string; border: string }>`
  - `severityMeta: Record<AlertSeverity, { color: string; dot: string }>`
  - `formatAge(s: number): string`, `formatPrice(p: number): string`, `formatUsd(n: number): string`

- [ ] **Step 1: Testleri yaz** (`lib/format.test.ts`)

```ts
import { scoreToLevel, formatAge, formatPrice, formatUsd, riskMeta } from "./format";

test("scoreToLevel boundaries", () => {
  expect(scoreToLevel(0)).toBe("critical");
  expect(scoreToLevel(24)).toBe("critical");
  expect(scoreToLevel(25)).toBe("high");
  expect(scoreToLevel(49)).toBe("high");
  expect(scoreToLevel(50)).toBe("medium");
  expect(scoreToLevel(69)).toBe("medium");
  expect(scoreToLevel(70)).toBe("good");
  expect(scoreToLevel(84)).toBe("good");
  expect(scoreToLevel(85)).toBe("strong");
  expect(scoreToLevel(100)).toBe("strong");
});

test("riskMeta has label+color for every level", () => {
  for (const lvl of ["critical", "high", "medium", "good", "strong"] as const) {
    expect(riskMeta[lvl].label).toBeTruthy();
    expect(riskMeta[lvl].color).toMatch(/^#/);
  }
});

test("formatAge seconds and minutes", () => {
  expect(formatAge(38)).toBe("38s");
  expect(formatAge(95)).toBe("1m 35s");
});

test("formatPrice tiers", () => {
  expect(formatPrice(1.5)).toBe("$1.50");
  expect(formatPrice(0.019)).toBe("$0.0190");
  expect(formatPrice(0.0000009)).toBe("$9.0e-7");
});

test("formatUsd tiers", () => {
  expect(formatUsd(320000)).toBe("$320.0K");
  expect(formatUsd(1_500_000)).toBe("$1.5M");
  expect(formatUsd(940)).toBe("$940");
});
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `npm run test -- format`
Expected: FAIL ("Cannot find module './format'").

- [ ] **Step 3: `lib/format.ts` implementasyonu** (referans `mock.ts` portu)

```ts
export type RiskLevel = "critical" | "high" | "medium" | "good" | "strong";
export type AlertSeverity = "info" | "positive" | "warning" | "critical";

export function scoreToLevel(score: number): RiskLevel {
  if (score <= 24) return "critical";
  if (score <= 49) return "high";
  if (score <= 69) return "medium";
  if (score <= 84) return "good";
  return "strong";
}

export const riskMeta: Record<RiskLevel, { label: string; color: string; bg: string; border: string }> = {
  critical: { label: "Critical", color: "#F0476B", bg: "rgba(240,71,107,0.12)", border: "rgba(240,71,107,0.35)" },
  high: { label: "High Risk", color: "#FFB020", bg: "rgba(255,176,32,0.12)", border: "rgba(255,176,32,0.35)" },
  medium: { label: "Medium", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)", border: "rgba(62,155,255,0.35)" },
  good: { label: "Good", color: "#2FD98B", bg: "rgba(47,217,139,0.12)", border: "rgba(47,217,139,0.35)" },
  strong: { label: "Strong", color: "#2FD98B", bg: "rgba(47,217,139,0.16)", border: "rgba(47,217,139,0.45)" },
};

export const severityMeta: Record<AlertSeverity, { color: string; dot: string }> = {
  info: { color: "#3E9BFF", dot: "#3E9BFF" },
  positive: { color: "#2FD98B", dot: "#2FD98B" },
  warning: { color: "#FFB020", dot: "#FFB020" },
  critical: { color: "#F0476B", dot: "#F0476B" },
};

export function formatAge(s: number): string {
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  return `${m}m ${s % 60}s`;
}

export function formatPrice(p: number): string {
  if (p >= 1) return `$${p.toFixed(2)}`;
  if (p >= 0.001) return `$${p.toFixed(4)}`;
  return `$${p.toExponential(1)}`;
}

export function formatUsd(n: number): string {
  if (n >= 1_000_000) return `$${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `$${(n / 1_000).toFixed(1)}K`;
  return `$${n}`;
}
```

- [ ] **Step 4: Testin geçtiğini doğrula**

Run: `npm run test -- format`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/format.ts apps/web/lib/format.test.ts
git commit -m "feat(web): add format utils and score-level logic with tests"
```

---

### Task 3: shadcn/ui primitives + `cn` util

**Files:**
- Create: `apps/web/lib/utils.ts` (`cn`)
- Create: `apps/web/components/ui/*` (shadcn generated)
- Test: `apps/web/components/ui/button.test.tsx`

**Interfaces:**
- Produces: `cn(...)`, `Button`, `Card`, `Badge`, `Tooltip*`, `ScrollArea`, `Separator`, `Skeleton`, `DropdownMenu*`, `Toaster` (sonner).

- [ ] **Step 1: `cn` util yaz** (`lib/utils.ts`)

```ts
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

- [ ] **Step 2: shadcn init + gerekli primitive'leri ekle**

```bash
npx shadcn@latest init -d
npx shadcn@latest add button card badge tooltip scroll-area separator skeleton dropdown-menu sonner
```

Not: shadcn `components/ui/*` üretir ve Tailwind v4 için `components.json` yazar. Üretilen dosyalar `@/lib/utils`'ten `cn` kullanır (Step 1 ile uyumlu).

- [ ] **Step 3: `button.test.tsx` — üretimin doğruluğu**

```tsx
import { render, screen } from "@testing-library/react";
import { Button } from "@/components/ui/button";

test("Button renders its children", () => {
  render(<Button>Trade</Button>);
  expect(screen.getByRole("button", { name: "Trade" })).toBeInTheDocument();
});
```

- [ ] **Step 4: Testi doğrula**

Run: `npm run test -- button`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/utils.ts apps/web/components/ui apps/web/components.json
git commit -m "feat(web): add shadcn/ui primitives and cn util"
```

---

### Task 4: Veri katmanı — types, contract, mock adapter, getApi (TDD)

**Files:**
- Create: `apps/web/lib/api/types.ts`, `apps/web/lib/api/contract.ts`, `apps/web/lib/api/mock.ts`, `apps/web/lib/api/http.ts`, `apps/web/lib/api/index.ts`
- Test: `apps/web/lib/api/mock.test.ts`, `apps/web/lib/api/index.test.ts`

**Interfaces:**
- Consumes: `RiskLevel`, `AlertSeverity` (Task 2 `lib/format.ts`).
- Produces:
  - `interface Kpi`, `interface TokenRow`, `interface AlertEvent`, `interface RadarPoint`
  - `interface SentinelApi { getKpis(); getTokens(); getAlerts(); getRadar(); subscribeTokens(cb); subscribeAlerts(cb); }`
  - `mockApi: SentinelApi`, `httpApi: SentinelApi` (stub), `getApi(): SentinelApi`

- [ ] **Step 1: `lib/api/types.ts` yaz**

```ts
import type { RiskLevel } from "@/lib/format";

export interface Kpi {
  id: string;
  label: string;
  value: string;
  change: number;
  spark: number[];
  updated: string;
  tone?: "positive" | "critical" | "warning" | "neutral";
}

export interface TokenRow {
  id: string;
  name: string;
  symbol: string;
  mint: string;
  ageSeconds: number;
  price: number;
  liquidity: number;
  vol5m: number;
  holders: number;
  creatorScore: number;
  safetyScore: number;
  momentum: number;
  spark: number[];
  signal: "buy" | "watch" | "avoid" | null;
  watchlisted: boolean;
}

export interface AlertEvent {
  id: string;
  type: string;
  token: string;
  detail: string;
  severity: import("@/lib/format").AlertSeverity;
  time: string;
}

export interface RadarPoint {
  x: number;
  y: number;
  z: number;
  name: string;
  level: RiskLevel;
}
```

- [ ] **Step 2: `lib/api/contract.ts` yaz**

```ts
import type { Kpi, TokenRow, AlertEvent, RadarPoint } from "./types";

export interface SentinelApi {
  getKpis(): Promise<Kpi[]>;
  getTokens(): Promise<TokenRow[]>;
  getAlerts(): Promise<AlertEvent[]>;
  getRadar(): Promise<RadarPoint[]>;
  /** Real-time seam — mock: interval, http: WebSocket. Returns unsubscribe fn. */
  subscribeTokens(cb: (tokens: TokenRow[]) => void): () => void;
  subscribeAlerts(cb: (alert: AlertEvent) => void): () => void;
}
```

- [ ] **Step 3: Mock testini yaz** (`lib/api/mock.test.ts`)

```ts
import { mockApi } from "./mock";

test("mockApi returns seed collections", async () => {
  expect((await mockApi.getKpis()).length).toBe(8);
  expect((await mockApi.getTokens()).length).toBeGreaterThan(0);
  expect((await mockApi.getAlerts()).length).toBeGreaterThan(0);
  expect((await mockApi.getRadar()).length).toBe((await mockApi.getTokens()).length);
});

test("subscribeTokens emits and returns an unsubscribe fn", async () => {
  await new Promise<void>((resolve, reject) => {
    const stop = mockApi.subscribeTokens((tokens) => {
      expect(Array.isArray(tokens)).toBe(true);
      stop();
      resolve();
    });
    expect(typeof stop).toBe("function");
    setTimeout(() => reject(new Error("no emit")), 4000);
  });
});
```

- [ ] **Step 4: Testin başarısız olduğunu doğrula**

Run: `npm run test -- api/mock`
Expected: FAIL (mock modülü yok).

- [ ] **Step 5: `lib/api/mock.ts` yaz** (referans `mock.ts` verisi + subscribe seam)

```ts
import type { SentinelApi } from "./contract";
import type { Kpi, TokenRow, AlertEvent, RadarPoint } from "./types";
import { scoreToLevel } from "@/lib/format";

function spark(seed: number, len = 16): number[] {
  const out: number[] = [];
  let v = 50 + (seed % 20);
  for (let i = 0; i < len; i++) {
    v += Math.sin(seed + i * 1.3) * 8 + ((seed * (i + 1)) % 7) - 3;
    out.push(Math.max(4, Math.round(v)));
  }
  return out;
}

const kpis: Kpi[] = [
  { id: "detected", label: "Tokens Detected (24h)", value: "3,412", change: 12.4, spark: spark(3), updated: "12s ago" },
  { id: "highconf", label: "High Confidence Tokens", value: "184", change: 8.1, spark: spark(7), updated: "12s ago", tone: "positive" },
  { id: "critical", label: "Critical Risk Detections", value: "97", change: 23.5, spark: spark(11), updated: "8s ago", tone: "critical" },
  { id: "signals", label: "Active Signals", value: "26", change: -4.2, spark: spark(5), updated: "3s ago" },
  { id: "positions", label: "Open Positions", value: "7", change: 0, spark: spark(9), updated: "1m ago" },
  { id: "realized", label: "Realized PnL (24h)", value: "+$4,182", change: 6.7, spark: spark(13), updated: "45s ago", tone: "positive" },
  { id: "unrealized", label: "Unrealized PnL", value: "-$612", change: -2.1, spark: spark(2), updated: "5s ago", tone: "warning" },
  { id: "latency", label: "System Latency", value: "142 ms", change: -11.0, spark: spark(6), updated: "2s ago" },
];

const tokens: TokenRow[] = [
  { id: "t1", name: "SolPulse", symbol: "PULSE", mint: "9xQeWv...4Fk2", ageSeconds: 38, price: 0.0042, liquidity: 82400, vol5m: 41200, holders: 312, creatorScore: 82, safetyScore: 78, momentum: 88, spark: spark(21), signal: "buy", watchlisted: true },
  { id: "t2", name: "NovaByte", symbol: "NOVA", mint: "Ap12Rd...9Zk1", ageSeconds: 95, price: 0.00011, liquidity: 15600, vol5m: 8900, holders: 96, creatorScore: 44, safetyScore: 52, momentum: 61, spark: spark(34), signal: "watch", watchlisted: false },
  { id: "t3", name: "GigaFrog", symbol: "GFROG", mint: "7mLp2c...1Qw8", ageSeconds: 12, price: 0.0000009, liquidity: 4200, vol5m: 22100, holders: 41, creatorScore: 17, safetyScore: 22, momentum: 74, spark: spark(48), signal: "avoid", watchlisted: false },
  { id: "t4", name: "Lumen", symbol: "LMN", mint: "Cd93Kf...6Rt4", ageSeconds: 210, price: 0.019, liquidity: 141000, vol5m: 63400, holders: 884, creatorScore: 91, safetyScore: 86, momentum: 79, spark: spark(15), signal: "buy", watchlisted: true },
  { id: "t5", name: "MoonCat", symbol: "MCAT", mint: "Ff01Xq...3Bn7", ageSeconds: 320, price: 0.0007, liquidity: 28900, vol5m: 12300, holders: 210, creatorScore: 63, safetyScore: 58, momentum: 55, spark: spark(27), signal: "watch", watchlisted: false },
  { id: "t6", name: "ZapDog", symbol: "ZAP", mint: "Gg44Lm...8Yu2", ageSeconds: 480, price: 0.0031, liquidity: 9800, vol5m: 3100, holders: 58, creatorScore: 29, safetyScore: 34, momentum: 42, spark: spark(39), signal: "avoid", watchlisted: false },
  { id: "t7", name: "Helios", symbol: "HLS", mint: "Hh77Nb...2Kl9", ageSeconds: 640, price: 0.11, liquidity: 320000, vol5m: 98000, holders: 1420, creatorScore: 88, safetyScore: 90, momentum: 71, spark: spark(19), signal: "buy", watchlisted: true },
  { id: "t8", name: "PixelFi", symbol: "PXL", mint: "Ii22Vc...5Dw3", ageSeconds: 830, price: 0.00054, liquidity: 46700, vol5m: 15600, holders: 340, creatorScore: 71, safetyScore: 66, momentum: 63, spark: spark(31), signal: "watch", watchlisted: false },
];

const alerts: AlertEvent[] = [
  { id: "a1", type: "Whale Buy", token: "PULSE", detail: "Wallet 4Fk2 bought 182 SOL", severity: "positive", time: "just now" },
  { id: "a2", type: "Liquidity Removed", token: "GFROG", detail: "Creator pulled 92% of pool", severity: "critical", time: "18s ago" },
  { id: "a3", type: "Score Change", token: "NOVA", detail: "Safety 62 → 52", severity: "warning", time: "44s ago" },
  { id: "a4", type: "First Liquidity", token: "LMN", detail: "$141K pool created", severity: "info", time: "1m ago" },
  { id: "a5", type: "Creator Sell", token: "ZAP", detail: "Creator sold 14% of supply", severity: "warning", time: "2m ago" },
  { id: "a6", type: "Trade Filled", token: "HLS", detail: "Buy 3.2 SOL @ 0.109", severity: "positive", time: "3m ago" },
  { id: "a7", type: "New Mint", token: "MCAT", detail: "Token created on Pump.fun", severity: "info", time: "4m ago" },
  { id: "a8", type: "Suspicious Cluster", token: "GFROG", detail: "5 linked wallets detected", severity: "critical", time: "5m ago" },
];

function radarFrom(list: TokenRow[]): RadarPoint[] {
  return list.map((t) => ({
    x: t.creatorScore,
    y: t.momentum,
    z: t.liquidity,
    name: t.symbol,
    level: scoreToLevel(Math.round((t.creatorScore + t.safetyScore) / 2)),
  }));
}

const delay = <T,>(v: T) => new Promise<T>((r) => setTimeout(() => r(v), 40));

export const mockApi: SentinelApi = {
  getKpis: () => delay(kpis),
  getTokens: () => delay(tokens),
  getAlerts: () => delay(alerts),
  getRadar: () => delay(radarFrom(tokens)),

  subscribeTokens(cb) {
    // Simulate live feed: nudge the youngest token's age/momentum every 2.5s.
    let live = tokens.map((t) => ({ ...t }));
    const id = setInterval(() => {
      live = live.map((t, i) =>
        i === 0 ? { ...t, ageSeconds: t.ageSeconds + 3, momentum: Math.min(100, t.momentum + 1) } : t
      );
      cb(live);
    }, 2500);
    return () => clearInterval(id);
  },

  subscribeAlerts(cb) {
    const pool: AlertEvent[] = [
      { id: "live1", type: "New Mint", token: "AERO", detail: "Token created on Raydium", severity: "info", time: "just now" },
      { id: "live2", type: "Whale Buy", token: "LMN", detail: "Wallet 6Rt4 bought 90 SOL", severity: "positive", time: "just now" },
    ];
    let n = 0;
    const id = setInterval(() => {
      cb({ ...pool[n % pool.length], id: `live-${Date.now()}` });
      n++;
    }, 5000);
    return () => clearInterval(id);
  },
};
```

- [ ] **Step 6: `lib/api/http.ts` stub yaz**

```ts
import type { SentinelApi } from "./contract";

// TODO(backend): AWS REST + WebSocket implementasyonu. Endpoint aileleri
// ROADMAP servislerine maplenir (tokens→discovery, alerts→alert engine, ...).
const notReady = () => Promise.reject(new Error("httpApi not implemented — backend not connected yet"));

export const httpApi: SentinelApi = {
  getKpis: notReady,
  getTokens: notReady,
  getAlerts: notReady,
  getRadar: notReady,
  subscribeTokens: () => () => {},
  subscribeAlerts: () => () => {},
};
```

- [ ] **Step 7: `lib/api/index.ts` — getApi()**

```ts
import type { SentinelApi } from "./contract";
import { mockApi } from "./mock";
import { httpApi } from "./http";

export function getApi(): SentinelApi {
  return process.env.NEXT_PUBLIC_DATA_SOURCE === "http" ? httpApi : mockApi;
}

export type { SentinelApi };
```

- [ ] **Step 8: `lib/api/index.test.ts` yaz**

```ts
import { getApi } from "./index";
import { mockApi } from "./mock";

test("getApi defaults to mock", () => {
  expect(getApi()).toBe(mockApi);
});
```

- [ ] **Step 9: Testleri doğrula**

Run: `npm run test -- api`
Expected: PASS (mock + index).

- [ ] **Step 10: Commit**

```bash
git add apps/web/lib/api
git commit -m "feat(web): add data layer contract, mock adapter, and getApi seam"
```

---

### Task 5: TanStack Query provider + hooks

**Files:**
- Create: `apps/web/app/providers.tsx`, `apps/web/lib/get-query-client.ts`, `apps/web/lib/hooks/queries.ts`
- Test: `apps/web/lib/hooks/queries.test.tsx`
- Modify: `apps/web/app/layout.tsx` (Providers ile sarma)

**Interfaces:**
- Consumes: `getApi()` (Task 4).
- Produces: `Providers`, `getQueryClient()`, query key sabitleri `qk`, `useKpis()`, `useTokens()`, `useAlerts()`, `useRadar()`.

- [ ] **Step 1: `lib/get-query-client.ts`** (server + client paylaşımlı)

```ts
import { QueryClient } from "@tanstack/react-query";

export function getQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { staleTime: 30_000, refetchOnWindowFocus: false } },
  });
}

export const qk = {
  kpis: ["kpis"] as const,
  tokens: ["tokens"] as const,
  alerts: ["alerts"] as const,
  radar: ["radar"] as const,
};
```

- [ ] **Step 2: `app/providers.tsx`** (client boundary)

```tsx
"use client";
import { useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";

export function Providers({ children }: { children: React.ReactNode }) {
  const [client] = useState(getQueryClient);
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
```

- [ ] **Step 3: `app/layout.tsx`'i Providers ile sar** — `<body>` içeriğini `<Providers>{children}</Providers>` ile değiştir; `import { Providers } from "./providers";` ekle.

- [ ] **Step 4: `lib/hooks/queries.ts`**

```ts
"use client";
import { useQuery } from "@tanstack/react-query";
import { getApi } from "@/lib/api";
import { qk } from "@/lib/get-query-client";

export function useKpis() {
  return useQuery({ queryKey: qk.kpis, queryFn: () => getApi().getKpis() });
}
export function useTokens() {
  return useQuery({ queryKey: qk.tokens, queryFn: () => getApi().getTokens() });
}
export function useAlerts() {
  return useQuery({ queryKey: qk.alerts, queryFn: () => getApi().getAlerts() });
}
export function useRadar() {
  return useQuery({ queryKey: qk.radar, queryFn: () => getApi().getRadar() });
}
```

- [ ] **Step 5: Hook testini yaz** (`lib/hooks/queries.test.tsx`)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { useKpis } from "./queries";

function Probe() {
  const { data } = useKpis();
  return <div>count:{data?.length ?? 0}</div>;
}

test("useKpis loads mock kpis", async () => {
  const client = getQueryClient();
  render(
    <QueryClientProvider client={client}>
      <Probe />
    </QueryClientProvider>
  );
  await waitFor(() => expect(screen.getByText("count:8")).toBeInTheDocument());
});
```

- [ ] **Step 6: Testi doğrula**

Run: `npm run test -- hooks/queries`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/providers.tsx apps/web/app/layout.tsx apps/web/lib/get-query-client.ts apps/web/lib/hooks
git commit -m "feat(web): add TanStack Query provider and data hooks"
```

---

### Task 6: Zustand stores (UI + session)

**Files:**
- Create: `apps/web/lib/store/ui.ts`, `apps/web/lib/store/session.ts`
- Test: `apps/web/lib/store/store.test.ts`

**Interfaces:**
- Produces:
  - `useUiStore` → `{ sidebarCollapsed: boolean; toggleSidebar(): void }`
  - `type TradingMode = "paper" | "shadow" | "live"`
  - `useSessionStore` → `{ tradingMode: TradingMode; setTradingMode(m: TradingMode): void }`

- [ ] **Step 1: Testi yaz** (`lib/store/store.test.ts`)

```ts
import { useUiStore } from "./ui";
import { useSessionStore } from "./session";

test("ui store toggles sidebar", () => {
  expect(useUiStore.getState().sidebarCollapsed).toBe(false);
  useUiStore.getState().toggleSidebar();
  expect(useUiStore.getState().sidebarCollapsed).toBe(true);
});

test("session store sets trading mode", () => {
  useSessionStore.getState().setTradingMode("live");
  expect(useSessionStore.getState().tradingMode).toBe("live");
});
```

- [ ] **Step 2: Testin başarısız olduğunu doğrula**

Run: `npm run test -- store`
Expected: FAIL (modüller yok).

- [ ] **Step 3: `lib/store/ui.ts`**

```ts
import { create } from "zustand";

interface UiState {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
}

export const useUiStore = create<UiState>((set) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
}));
```

- [ ] **Step 4: `lib/store/session.ts`**

```ts
import { create } from "zustand";

export type TradingMode = "paper" | "shadow" | "live";

interface SessionState {
  tradingMode: TradingMode;
  setTradingMode: (m: TradingMode) => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  tradingMode: "paper",
  setTradingMode: (m) => set({ tradingMode: m }),
}));
```

- [ ] **Step 5: Testi doğrula**

Run: `npm run test -- store`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/web/lib/store
git commit -m "feat(web): add zustand ui and session stores"
```

---

### Task 7: Sentinel primitive'leri (ScoreBadge, Sparkline, TokenAvatar, WalletAddress, KpiCard)

**Files:**
- Create: `apps/web/components/sentinel/Sparkline.tsx`, `ScoreBadge.tsx`, `TokenAvatar.tsx`, `WalletAddress.tsx`, `KpiCard.tsx`
- Test: `apps/web/components/sentinel/ScoreBadge.test.tsx`, `WalletAddress.test.tsx`

**Interfaces:**
- Consumes: `scoreToLevel`, `riskMeta` (format), `Kpi` (api/types).
- Produces: `<Sparkline data color width height/>`, `<ScoreBadge score label showBar size/>`, `<TokenAvatar symbol size/>`, `<WalletAddress address explorer/>`, `<KpiCard kpi/>`.

- [ ] **Step 1: `Sparkline.tsx`** (referans portu; SSR-safe, `"use client"` gerekmez — saf SVG)

```tsx
interface SparklineProps { data: number[]; color?: string; width?: number; height?: number; }

export function Sparkline({ data, color = "#7C5CFF", width = 88, height = 28 }: SparklineProps) {
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const step = width / (data.length - 1);
  const pts = data.map((v, i) => `${i * step},${height - ((v - min) / range) * height}`);
  const path = `M ${pts.join(" L ")}`;
  const areaId = `spark-${color.replace("#", "")}`;
  return (
    <svg width={width} height={height} className="overflow-visible">
      <defs>
        <linearGradient id={areaId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity={0.25} />
          <stop offset="100%" stopColor={color} stopOpacity={0} />
        </linearGradient>
      </defs>
      <path d={`${path} L ${width},${height} L 0,${height} Z`} fill={`url(#${areaId})`} />
      <path d={path} fill="none" stroke={color} strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
```

- [ ] **Step 2: `ScoreBadge.tsx`** (import kaynağı `@/lib/format`)

```tsx
import { scoreToLevel, riskMeta } from "@/lib/format";

interface ScoreBadgeProps { score: number; label?: string; showBar?: boolean; size?: "sm" | "md"; }

export function ScoreBadge({ score, label, showBar = false, size = "sm" }: ScoreBadgeProps) {
  const level = scoreToLevel(score);
  const meta = riskMeta[level];
  return (
    <div className="inline-flex flex-col gap-1">
      <div
        className="inline-flex items-center gap-1.5 rounded-md px-2 py-0.5"
        style={{ backgroundColor: meta.bg, border: `1px solid ${meta.border}` }}
        title={label ? `${label}: ${meta.label}` : meta.label}
      >
        <span className="font-mono tabular-nums" style={{ color: meta.color, fontSize: size === "sm" ? 12 : 14, fontWeight: 600 }}>
          {score}
        </span>
        <span style={{ color: meta.color, fontSize: 11, opacity: 0.85 }}>{meta.label}</span>
      </div>
      {showBar && (
        <div className="h-1 w-full overflow-hidden rounded-full bg-surface-3">
          <div className="h-full rounded-full" style={{ width: `${score}%`, backgroundColor: meta.color }} />
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: `TokenAvatar.tsx`** (referans portu, aynen)

```tsx
interface TokenAvatarProps { symbol: string; size?: number; }
const palette = ["#7C5CFF", "#3E9BFF", "#2FD98B", "#FFB020", "#F0476B", "#9B6BFF"];

export function TokenAvatar({ symbol, size = 28 }: TokenAvatarProps) {
  const hash = symbol.split("").reduce((a, c) => a + c.charCodeAt(0), 0);
  const bg = palette[hash % palette.length];
  return (
    <div
      className="flex shrink-0 items-center justify-center rounded-full font-mono"
      style={{ width: size, height: size, fontSize: size * 0.38, fontWeight: 600, color: "#0B1019", background: `linear-gradient(135deg, ${bg}, ${bg}bb)` }}
    >
      {symbol.slice(0, 2)}
    </div>
  );
}
```

- [ ] **Step 4: `WalletAddress.tsx`** (`"use client"`; copy → sonner toast)

```tsx
"use client";
import { useState } from "react";
import { Copy, Check, ExternalLink } from "lucide-react";
import { toast } from "sonner";

interface WalletAddressProps { address: string; explorer?: boolean; }

export function WalletAddress({ address, explorer = true }: WalletAddressProps) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(address);
    setCopied(true);
    toast.success("Address copied", { description: address });
    setTimeout(() => setCopied(false), 1400);
  };
  return (
    <span className="group inline-flex items-center gap-1.5">
      <span className="font-mono text-muted-foreground" style={{ fontSize: 12 }}>{address}</span>
      <button onClick={copy} className="text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100" title="Copy address">
        {copied ? <Check size={13} className="text-positive" /> : <Copy size={13} />}
      </button>
      {explorer && (
        <a href="#" onClick={(e) => e.preventDefault()} className="text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100" title="View on explorer">
          <ExternalLink size={13} />
        </a>
      )}
    </span>
  );
}
```

- [ ] **Step 5: `KpiCard.tsx`** (referans portu; `Kpi` tipi `@/lib/api/types`'ten)

```tsx
import { ArrowUpRight, ArrowDownRight, Minus } from "lucide-react";
import { Sparkline } from "./Sparkline";
import type { Kpi } from "@/lib/api/types";

const toneColor: Record<NonNullable<Kpi["tone"]> | "default", string> = {
  positive: "#2FD98B", critical: "#F0476B", warning: "#FFB020", neutral: "#8A94A6", default: "#7C5CFF",
};

export function KpiCard({ kpi }: { kpi: Kpi }) {
  const color = toneColor[kpi.tone ?? "default"];
  const up = kpi.change > 0;
  const flat = kpi.change === 0;
  const changeColor = flat ? "#8A94A6" : up ? "#2FD98B" : "#F0476B";
  const ChangeIcon = flat ? Minus : up ? ArrowUpRight : ArrowDownRight;
  return (
    <div className="group rounded-lg border border-border bg-card p-4 transition-colors hover:border-white/15" title={`${kpi.label} — updated ${kpi.updated}`}>
      <div className="flex items-start justify-between gap-2">
        <span className="text-muted-foreground" style={{ fontSize: 12 }}>{kpi.label}</span>
        <span className="inline-flex items-center gap-0.5 font-mono tabular-nums" style={{ color: changeColor, fontSize: 11 }}>
          <ChangeIcon size={12} />{flat ? "0.0" : Math.abs(kpi.change).toFixed(1)}%
        </span>
      </div>
      <div className="mt-2 flex items-end justify-between gap-2">
        <span className="font-mono tabular-nums" style={{ fontSize: 22, fontWeight: 600, color }}>{kpi.value}</span>
        <Sparkline data={kpi.spark} color={color} />
      </div>
      <div className="mt-1.5 text-muted-foreground" style={{ fontSize: 10 }}>Updated {kpi.updated}</div>
    </div>
  );
}
```

- [ ] **Step 6: `ScoreBadge.test.tsx`**

```tsx
import { render, screen } from "@testing-library/react";
import { ScoreBadge } from "./ScoreBadge";

test("ScoreBadge shows number and level label", () => {
  render(<ScoreBadge score={17} />);
  expect(screen.getByText("17")).toBeInTheDocument();
  expect(screen.getByText("Critical")).toBeInTheDocument();
});

test("ScoreBadge maps 88 to Strong", () => {
  render(<ScoreBadge score={88} />);
  expect(screen.getByText("Strong")).toBeInTheDocument();
});
```

- [ ] **Step 7: `WalletAddress.test.tsx`** (copy → toast)

```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WalletAddress } from "./WalletAddress";

test("copy writes to clipboard", async () => {
  const writeText = vi.fn();
  Object.assign(navigator, { clipboard: { writeText } });
  render(<WalletAddress address="9xQeWv...4Fk2" />);
  await userEvent.click(screen.getByTitle("Copy address"));
  expect(writeText).toHaveBeenCalledWith("9xQeWv...4Fk2");
});
```

- [ ] **Step 8: Testleri doğrula**

Run: `npm run test -- sentinel`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add apps/web/components/sentinel
git commit -m "feat(web): add Sentinel primitives (score, sparkline, avatar, address, kpi)"
```

---

### Task 8: App shell (nav, TradingModeBadge, Sidebar, Header) + route grubu + placeholder + tüm route'lar

**Files:**
- Create: `apps/web/components/shell/nav.ts`, `TradingModeBadge.tsx`, `Sidebar.tsx`, `Header.tsx`
- Create: `apps/web/components/PlaceholderScreen.tsx`
- Create: `apps/web/app/(app)/layout.tsx`
- Create: 16 placeholder route: `apps/web/app/(app)/<path>/page.tsx`
- Modify: `apps/web/app/(app)/page.tsx` (geçici Overview — Task 9'da doldurulacak)
- Delete: `apps/web/app/page.tsx` (kök geçici sayfa → `(app)/page.tsx`'e taşınır)
- Test: `apps/web/components/shell/Sidebar.test.tsx`, `TradingModeBadge.test.tsx`

**Interfaces:**
- Consumes: `useUiStore`, `useSessionStore`/`TradingMode` (Task 6), `navItems`.
- Produces: `navItems: NavItem[]`, `<Sidebar/>`, `<Header/>`, `<TradingModeBadge/>`, `<PlaceholderScreen/>`, `(app)` shell layout.

- [ ] **Step 1: `components/shell/nav.ts`** (referans portu)

```ts
import {
  LayoutDashboard, Radio, Compass, Coins, UserSearch, Share2, Sparkles,
  Layers, Briefcase, ListOrdered, PieChart, History, Bell, Send, Bot,
  Activity, Settings, type LucideIcon,
} from "lucide-react";

export interface NavItem { label: string; path: string; icon: LucideIcon; }

export const navItems: NavItem[] = [
  { label: "Overview", path: "/", icon: LayoutDashboard },
  { label: "Live Feed", path: "/live-feed", icon: Radio },
  { label: "Discover", path: "/discover", icon: Compass },
  { label: "Tokens", path: "/tokens", icon: Coins },
  { label: "Creators", path: "/creators", icon: UserSearch },
  { label: "Wallet Graph", path: "/wallet-graph", icon: Share2 },
  { label: "Smart Wallets", path: "/smart-wallets", icon: Sparkles },
  { label: "Strategies", path: "/strategies", icon: Layers },
  { label: "Positions", path: "/positions", icon: Briefcase },
  { label: "Orders", path: "/orders", icon: ListOrdered },
  { label: "Portfolio", path: "/portfolio", icon: PieChart },
  { label: "Backtesting", path: "/backtesting", icon: History },
  { label: "Alerts", path: "/alerts", icon: Bell },
  { label: "Telegram", path: "/telegram", icon: Send },
  { label: "Research", path: "/research", icon: Bot },
  { label: "System Health", path: "/system-health", icon: Activity },
  { label: "Settings", path: "/settings", icon: Settings },
];
```

- [ ] **Step 2: `components/shell/TradingModeBadge.tsx`** (`"use client"`; Zustand ile bağlı)

```tsx
"use client";
import { useState } from "react";
import { FlaskConical, Eye, Zap, ChevronDown } from "lucide-react";
import { useSessionStore, type TradingMode } from "@/lib/store/session";

const modeMeta: Record<TradingMode, { label: string; color: string; bg: string; icon: typeof Zap }> = {
  paper: { label: "Paper", color: "#3E9BFF", bg: "rgba(62,155,255,0.14)", icon: FlaskConical },
  shadow: { label: "Shadow", color: "#FFB020", bg: "rgba(255,176,32,0.14)", icon: Eye },
  live: { label: "Live", color: "#F0476B", bg: "rgba(240,71,107,0.16)", icon: Zap },
};

export function TradingModeBadge({ collapsed }: { collapsed?: boolean }) {
  const mode = useSessionStore((s) => s.tradingMode);
  const onChange = useSessionStore((s) => s.setTradingMode);
  const [open, setOpen] = useState(false);
  const meta = modeMeta[mode];
  const Icon = meta.icon;
  return (
    <div className="relative">
      <button onClick={() => setOpen((o) => !o)} className="flex w-full items-center gap-2 rounded-md px-2.5 py-2"
        style={{ backgroundColor: meta.bg, border: `1px solid ${meta.color}55` }} title="Trading mode">
        <Icon size={15} style={{ color: meta.color }} />
        {!collapsed && (
          <>
            <span className="flex flex-col items-start leading-tight">
              <span className="text-muted-foreground" style={{ fontSize: 9 }}>MODE</span>
              <span style={{ fontSize: 12, fontWeight: 600, color: meta.color }}>{meta.label} Trading</span>
            </span>
            <ChevronDown size={14} className="ml-auto text-muted-foreground" />
          </>
        )}
      </button>
      {open && (
        <div className="absolute bottom-full left-0 z-20 mb-2 w-full min-w-[180px] rounded-md border border-border bg-popover p-1 shadow-xl">
          {(Object.keys(modeMeta) as TradingMode[]).map((m) => {
            const mm = modeMeta[m];
            const MIcon = mm.icon;
            return (
              <button key={m} onClick={() => { onChange(m); setOpen(false); }} className="flex w-full items-center gap-2 rounded px-2 py-1.5 hover:bg-accent">
                <MIcon size={14} style={{ color: mm.color }} />
                <span style={{ fontSize: 13 }}>{mm.label}</span>
                {m === "live" && (
                  <span className="ml-auto rounded px-1.5 py-0.5" style={{ fontSize: 9, color: "#F0476B", backgroundColor: "rgba(240,71,107,0.14)" }}>REAL FUNDS</span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 3: `components/shell/Sidebar.tsx`** (`"use client"`; react-router `NavLink` → Next `Link`+`usePathname`, Zustand collapse)

```tsx
"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronLeft, ShieldCheck, Wifi, Send } from "lucide-react";
import { navItems } from "./nav";
import { TradingModeBadge } from "./TradingModeBadge";
import { useUiStore } from "@/lib/store/ui";

export function Sidebar() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const onToggle = useUiStore((s) => s.toggleSidebar);
  const pathname = usePathname();
  return (
    <aside className="flex h-full flex-col border-r border-sidebar-border bg-sidebar transition-all duration-200" style={{ width: collapsed ? 68 : 232 }}>
      <div className="flex h-14 items-center gap-2.5 border-b border-sidebar-border px-4">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary">
          <ShieldCheck size={18} className="text-primary-foreground" />
        </div>
        {!collapsed && (
          <div className="flex flex-col leading-tight">
            <span style={{ fontSize: 15, fontWeight: 600 }}>Sentinel</span>
            <span className="text-muted-foreground" style={{ fontSize: 10 }}>Solana Intelligence</span>
          </div>
        )}
        <button onClick={onToggle} className="ml-auto text-muted-foreground transition-colors hover:text-foreground" title={collapsed ? "Expand" : "Collapse"}>
          <ChevronLeft size={16} className={collapsed ? "rotate-180" : ""} />
        </button>
      </div>
      <nav className="flex-1 overflow-y-auto px-2 py-3">
        {navItems.map((item) => {
          const isActive = item.path === "/" ? pathname === "/" : pathname.startsWith(item.path);
          const Icon = item.icon;
          return (
            <Link key={item.path} href={item.path} title={collapsed ? item.label : undefined}
              className={`flex items-center gap-3 rounded-md px-2.5 py-2 transition-colors ${isActive ? "bg-sidebar-accent text-sidebar-accent-foreground" : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-foreground"}`}>
              <Icon size={17} className={isActive ? "text-primary" : ""} />
              {!collapsed && <span style={{ fontSize: 13 }}>{item.label}</span>}
            </Link>
          );
        })}
      </nav>
      <div className="border-t border-sidebar-border p-3">
        {!collapsed && (
          <div className="mb-3 space-y-1.5">
            <StatusRow icon={<Wifi size={12} />} label="RPC" value="142 ms" ok />
            <StatusRow icon={<span className="h-2 w-2 rounded-full bg-positive" />} label="Solana" value="Healthy" ok />
            <StatusRow icon={<Send size={12} />} label="Telegram" value="Connected" ok />
          </div>
        )}
        <TradingModeBadge collapsed={collapsed} />
      </div>
    </aside>
  );
}

function StatusRow({ icon, label, value, ok }: { icon: React.ReactNode; label: string; value: string; ok?: boolean }) {
  return (
    <div className="flex items-center gap-2 text-muted-foreground" style={{ fontSize: 11 }}>
      <span className={ok ? "text-positive" : "text-critical"}>{icon}</span>
      <span>{label}</span>
      <span className="ml-auto font-mono text-foreground">{value}</span>
    </div>
  );
}
```

- [ ] **Step 4: `components/shell/Header.tsx`** (`"use client"`; Live indicator sessionStore'dan)

```tsx
"use client";
import { Search, Bell, Pause, Gauge, Fuel } from "lucide-react";
import { useSessionStore } from "@/lib/store/session";

export function Header() {
  const mode = useSessionStore((s) => s.tradingMode);
  return (
    <header className="relative flex h-14 shrink-0 items-center gap-4 border-b border-border bg-background/80 px-5 backdrop-blur">
      <div className="relative w-full max-w-md">
        <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
        <input placeholder="Token, wallet, creator veya transaction ara"
          className="h-9 w-full rounded-md border border-border bg-input pl-9 pr-16 text-foreground placeholder:text-muted-foreground focus:border-primary focus:outline-none" style={{ fontSize: 13 }} />
        <kbd className="absolute right-3 top-1/2 -translate-y-1/2 rounded border border-border px-1.5 py-0.5 font-mono text-muted-foreground" style={{ fontSize: 10 }}>⌘K</kbd>
      </div>
      <div className="ml-auto flex items-center gap-4">
        <select className="hidden h-8 rounded-md border border-border bg-input px-2 text-foreground focus:outline-none md:block" style={{ fontSize: 12 }} defaultValue="mainnet">
          <option value="mainnet">Solana Mainnet</option>
          <option value="devnet">Devnet</option>
        </select>
        <div className="hidden items-center gap-3 lg:flex">
          <Metric icon={<Gauge size={13} />} label="RPC" value="142ms" color="#2FD98B" />
          <Metric icon={<Fuel size={13} />} label="Fee" value="0.00021" color="#8A94A6" />
          <span className="text-muted-foreground" style={{ fontSize: 11 }}>Updated 12s ago</span>
        </div>
        <button className="flex h-8 items-center gap-1.5 rounded-md px-3 transition-colors"
          style={{ backgroundColor: "rgba(240,71,107,0.12)", border: "1px solid rgba(240,71,107,0.4)", color: "#F0476B", fontSize: 12, fontWeight: 600 }}
          title="Halt all automated trading immediately">
          <Pause size={14} />Emergency Pause
        </button>
        <button className="relative text-muted-foreground transition-colors hover:text-foreground" title="Notifications">
          <Bell size={18} />
          <span className="absolute -right-0.5 -top-0.5 h-2 w-2 rounded-full bg-critical" />
        </button>
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary font-mono" style={{ fontSize: 12, fontWeight: 600, color: "#fff" }}>AK</div>
      </div>
      {mode === "live" && (
        <div className="pointer-events-none absolute left-0 top-14 z-30 h-0.5 w-full" style={{ background: "linear-gradient(90deg,#F0476B,transparent)" }} />
      )}
    </header>
  );
}

function Metric({ icon, label, value, color }: { icon: React.ReactNode; label: string; value: string; color: string }) {
  return (
    <span className="flex items-center gap-1 text-muted-foreground" style={{ fontSize: 11 }}>
      <span style={{ color }}>{icon}</span>{label} <span className="font-mono text-foreground">{value}</span>
    </span>
  );
}
```

- [ ] **Step 5: `components/PlaceholderScreen.tsx`** (`"use client"`; `usePathname` ile başlık)

```tsx
"use client";
import { usePathname } from "next/navigation";
import { Construction } from "lucide-react";
import { navItems } from "@/components/shell/nav";

export function PlaceholderScreen() {
  const pathname = usePathname();
  const item = navItems.find((n) => n.path === pathname);
  const Icon = item?.icon ?? Construction;
  return (
    <div className="space-y-5">
      <div>
        <h1>{item?.label ?? "Screen"}</h1>
        <p className="text-muted-foreground" style={{ fontSize: 13 }}>This module is part of the Sentinel platform blueprint.</p>
      </div>
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border bg-card py-24 text-center">
        <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-accent">
          <Icon size={26} className="text-primary" />
        </div>
        <h3 className="mb-1">{item?.label} coming soon</h3>
        <p className="mb-5 max-w-sm text-muted-foreground" style={{ fontSize: 13 }}>
          The Overview Dashboard is fully built out. Tell me which screen to detail next.
        </p>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: `app/(app)/layout.tsx`** (server component shell + Toaster)

```tsx
import { Sidebar } from "@/components/shell/Sidebar";
import { Header } from "@/components/shell/Header";
import { Toaster } from "@/components/ui/sonner";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen w-full overflow-hidden bg-background text-foreground">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <div className="mx-auto max-w-[1440px]">{children}</div>
        </main>
      </div>
      <Toaster theme="dark" position="bottom-right" />
    </div>
  );
}
```

- [ ] **Step 7: Kök geçici sayfayı taşı** — `apps/web/app/page.tsx`'i sil; `apps/web/app/(app)/page.tsx` oluştur (geçici):

```tsx
export default function OverviewPage() {
  return <div><h1>Overview</h1></div>;
}
```

- [ ] **Step 8: 16 placeholder route oluştur** — her `nav.ts` path'i (`/` hariç) için `app/(app)/<path>/page.tsx`. Örnek (`live-feed`):

```tsx
import { PlaceholderScreen } from "@/components/PlaceholderScreen";
export default function Page() { return <PlaceholderScreen />; }
```

Aynı içerik şu dizinlerde: `live-feed, discover, tokens, creators, wallet-graph, smart-wallets, strategies, positions, orders, portfolio, backtesting, alerts, telegram, research, system-health, settings`.

- [ ] **Step 9: Testleri yaz** (`Sidebar.test.tsx`, `TradingModeBadge.test.tsx`)

```tsx
// Sidebar.test.tsx
import { render, screen } from "@testing-library/react";
import { Sidebar } from "./Sidebar";
vi.mock("next/navigation", () => ({ usePathname: () => "/tokens" }));

test("Sidebar renders all nav items and marks active", () => {
  render(<Sidebar />);
  expect(screen.getByText("Overview")).toBeInTheDocument();
  const tokens = screen.getByText("Tokens").closest("a")!;
  expect(tokens.className).toContain("bg-sidebar-accent");
});
```

```tsx
// TradingModeBadge.test.tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TradingModeBadge } from "./TradingModeBadge";
import { useSessionStore } from "@/lib/store/session";

test("selecting Live updates the session store", async () => {
  useSessionStore.setState({ tradingMode: "paper" });
  render(<TradingModeBadge />);
  await userEvent.click(screen.getByTitle("Trading mode"));
  await userEvent.click(screen.getByText("Live"));
  expect(useSessionStore.getState().tradingMode).toBe("live");
});
```

- [ ] **Step 10: Testleri ve build'i doğrula**

Run: `npm run test -- shell`
Expected: PASS.
Run: `npm run build`
Expected: tüm route'lar derleniyor, hata yok.

- [ ] **Step 11: Commit**

```bash
git add apps/web/components/shell apps/web/components/PlaceholderScreen.tsx "apps/web/app/(app)"
git rm apps/web/app/page.tsx
git commit -m "feat(web): add app shell, nav, placeholders, and all routes"
```

---

### Task 9: Overview dashboard bileşenleri + sayfa (RSC prefetch + hydration)

**Files:**
- Create: `apps/web/components/dashboard/LiveTokenFeed.tsx`, `OpportunityRadar.tsx`, `AlertsTimeline.tsx`
- Create: `apps/web/components/dashboard/OverviewContent.tsx`
- Modify: `apps/web/app/(app)/page.tsx` (RSC prefetch + HydrationBoundary)
- Test: `apps/web/components/dashboard/LiveTokenFeed.test.tsx`

**Interfaces:**
- Consumes: `useKpis/useTokens/useAlerts/useRadar` (Task 5), Sentinel primitives (Task 7), `formatAge/formatPrice/formatUsd/riskMeta/severityMeta` (Task 2), `getQueryClient/qk` (Task 5).
- Produces: `<LiveTokenFeed/>`, `<OpportunityRadar/>`, `<AlertsTimeline/>`, `<OverviewContent/>`, dolu Overview sayfası.

- [ ] **Step 1: `LiveTokenFeed.tsx`** (`"use client"`; veri `useTokens`, sort local state)

```tsx
"use client";
import { useState } from "react";
import { Star, Activity, ShoppingCart, ArrowUpDown } from "lucide-react";
import { toast } from "sonner";
import { useTokens } from "@/lib/hooks/queries";
import { formatAge, formatPrice, formatUsd } from "@/lib/format";
import type { TokenRow } from "@/lib/api/types";
import { TokenAvatar } from "@/components/sentinel/TokenAvatar";
import { ScoreBadge } from "@/components/sentinel/ScoreBadge";
import { WalletAddress } from "@/components/sentinel/WalletAddress";
import { Sparkline } from "@/components/sentinel/Sparkline";

const signalMeta: Record<NonNullable<TokenRow["signal"]>, { label: string; color: string; bg: string }> = {
  buy: { label: "Buy", color: "#2FD98B", bg: "rgba(47,217,139,0.12)" },
  watch: { label: "Watch", color: "#3E9BFF", bg: "rgba(62,155,255,0.12)" },
  avoid: { label: "Avoid", color: "#F0476B", bg: "rgba(240,71,107,0.12)" },
};
type SortKey = "ageSeconds" | "liquidity" | "momentum" | "creatorScore";

export function LiveTokenFeed() {
  const { data } = useTokens();
  const [watch, setWatch] = useState<Record<string, boolean>>({});
  const [sortKey, setSortKey] = useState<SortKey>("ageSeconds");
  const rows = data ?? [];
  const sorted = [...rows].sort((a, b) => (sortKey === "ageSeconds" ? a[sortKey] - b[sortKey] : b[sortKey] - a[sortKey]));
  const isWatched = (t: TokenRow) => watch[t.id] ?? t.watchlisted;
  const toggle = (id: string) => setWatch((w) => ({ ...w, [id]: !(w[id] ?? rows.find((r) => r.id === id)?.watchlisted) }));

  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-positive opacity-60" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-positive" />
          </span>
          <h3>Live Token Feed</h3>
          <span className="text-muted-foreground" style={{ fontSize: 12 }}>· real-time</span>
        </div>
        <div className="flex items-center gap-1">
          {(["ageSeconds", "liquidity", "momentum", "creatorScore"] as SortKey[]).map((k) => (
            <button key={k} onClick={() => setSortKey(k)}
              className={`flex items-center gap-1 rounded px-2 py-1 ${sortKey === k ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground"}`}
              style={{ fontSize: 11 }}>
              <ArrowUpDown size={11} />
              {k === "ageSeconds" ? "Age" : k === "liquidity" ? "Liq" : k === "momentum" ? "Momentum" : "Creator"}
            </button>
          ))}
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead>
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {["Token", "Age", "Price", "Liquidity", "5m Vol", "Holders", "Creator", "Safety", "Momentum", "Signal", ""].map((h) => (
                <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sorted.map((t) => {
              const sig = t.signal ? signalMeta[t.signal] : null;
              const fresh = t.ageSeconds < 60;
              return (
                <tr key={t.id} data-fresh={fresh} className="border-t border-border transition-colors hover:bg-accent/40"
                  style={fresh ? { boxShadow: "inset 2px 0 0 #2FD98B" } : undefined}>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-2.5">
                      <TokenAvatar symbol={t.symbol} />
                      <div className="flex flex-col leading-tight">
                        <span style={{ fontWeight: 500 }}>{t.name} <span className="text-muted-foreground">{t.symbol}</span></span>
                        <WalletAddress address={t.mint} />
                      </div>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums" style={{ color: fresh ? "#2FD98B" : undefined }}>{formatAge(t.ageSeconds)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{formatPrice(t.price)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{formatUsd(t.liquidity)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{formatUsd(t.vol5m)}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono tabular-nums">{t.holders}</td>
                  <td className="px-3 py-2.5"><ScoreBadge score={t.creatorScore} label="Creator" /></td>
                  <td className="px-3 py-2.5"><ScoreBadge score={t.safetyScore} label="Safety" /></td>
                  <td className="px-3 py-2.5">
                    <Sparkline data={t.spark} color={t.momentum >= 70 ? "#2FD98B" : t.momentum >= 50 ? "#3E9BFF" : "#F0476B"} width={64} height={22} />
                  </td>
                  <td className="px-3 py-2.5">
                    {sig && <span className="rounded px-2 py-0.5" style={{ color: sig.color, backgroundColor: sig.bg, fontSize: 11, fontWeight: 600 }}>{sig.label}</span>}
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-1">
                      <button onClick={() => toggle(t.id)} className="rounded p-1 hover:bg-accent" title="Watchlist">
                        <Star size={14} className={isWatched(t) ? "fill-warning text-warning" : "text-muted-foreground"} />
                      </button>
                      <button onClick={() => toast("Analyze " + t.symbol)} className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground" title="Analyze"><Activity size={14} /></button>
                      <button onClick={() => toast("Trade " + t.symbol)} className="rounded p-1 text-primary hover:bg-accent" title="Trade"><ShoppingCart size={14} /></button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: `OpportunityRadar.tsx`** (`"use client"`; veri `useRadar`)

```tsx
"use client";
import { ScatterChart, Scatter, XAxis, YAxis, ZAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from "recharts";
import { useRadar } from "@/lib/hooks/queries";
import { riskMeta } from "@/lib/format";

export function OpportunityRadar() {
  const { data } = useRadar();
  const radarData = data ?? [];
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3>Opportunity Radar</h3>
        <span className="text-muted-foreground" style={{ fontSize: 11 }}>Creator Trust × Momentum · size = liquidity</span>
      </div>
      <div className="flex-1 p-3" style={{ minHeight: 260 }}>
        <ResponsiveContainer width="100%" height="100%">
          <ScatterChart margin={{ top: 10, right: 10, bottom: 20, left: 0 }}>
            <CartesianGrid stroke="rgba(255,255,255,0.05)" />
            <XAxis type="number" dataKey="x" name="Creator Trust" domain={[0, 100]} tick={{ fill: "#8A94A6", fontSize: 11 }} stroke="rgba(255,255,255,0.1)"
              label={{ value: "Creator Trust Score", position: "insideBottom", offset: -12, fill: "#8A94A6", fontSize: 11 }} />
            <YAxis type="number" dataKey="y" name="Momentum" domain={[0, 100]} tick={{ fill: "#8A94A6", fontSize: 11 }} stroke="rgba(255,255,255,0.1)"
              label={{ value: "Momentum", angle: -90, position: "insideLeft", fill: "#8A94A6", fontSize: 11 }} />
            <ZAxis type="number" dataKey="z" range={[80, 600]} />
            <Tooltip cursor={{ strokeDasharray: "3 3", stroke: "rgba(255,255,255,0.2)" }}
              contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }}
              formatter={(v: number, n: string) => [v, n]} labelFormatter={() => ""} />
            <Scatter data={radarData}>
              {radarData.map((d, i) => (<Cell key={i} fill={riskMeta[d.level].color} fillOpacity={0.75} />))}
            </Scatter>
          </ScatterChart>
        </ResponsiveContainer>
      </div>
      <div className="flex flex-wrap gap-3 border-t border-border px-4 py-2.5">
        {(["strong", "good", "medium", "high", "critical"] as const).map((l) => (
          <span key={l} className="flex items-center gap-1.5 text-muted-foreground" style={{ fontSize: 11 }}>
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: riskMeta[l].color }} />{riskMeta[l].label}
          </span>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 3: `AlertsTimeline.tsx`** (`"use client"`; veri `useAlerts`)

```tsx
"use client";
import { useAlerts } from "@/lib/hooks/queries";
import { severityMeta } from "@/lib/format";

export function AlertsTimeline() {
  const { data } = useAlerts();
  const alerts = data ?? [];
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3>Alerts Timeline</h3>
        <button className="text-primary" style={{ fontSize: 12 }}>View all</button>
      </div>
      <div className="flex-1 overflow-y-auto p-3">
        <ol className="relative ml-1 space-y-3 border-l border-border pl-4">
          {alerts.map((a) => {
            const meta = severityMeta[a.severity];
            return (
              <li key={a.id} className="relative">
                <span className="absolute -left-[21px] top-1 h-2.5 w-2.5 rounded-full ring-4 ring-card" style={{ backgroundColor: meta.dot }} />
                <div className="flex items-center justify-between gap-2">
                  <span style={{ fontSize: 13, fontWeight: 500, color: meta.color }}>{a.type}</span>
                  <span className="text-muted-foreground" style={{ fontSize: 11 }}>{a.time}</span>
                </div>
                <div className="text-muted-foreground" style={{ fontSize: 12 }}>
                  <span className="font-mono text-foreground">{a.token}</span> · {a.detail}
                </div>
              </li>
            );
          })}
        </ol>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: `OverviewContent.tsx`** (`"use client"`; KPI grid + kompozisyon)

```tsx
"use client";
import { useKpis } from "@/lib/hooks/queries";
import { KpiCard } from "@/components/sentinel/KpiCard";
import { LiveTokenFeed } from "./LiveTokenFeed";
import { OpportunityRadar } from "./OpportunityRadar";
import { AlertsTimeline } from "./AlertsTimeline";

export function OverviewContent() {
  const { data: kpis } = useKpis();
  return (
    <div className="space-y-5">
      <div>
        <h1>Overview</h1>
        <p className="text-muted-foreground" style={{ fontSize: 13 }}>
          Real-time Solana token intelligence · Discover → Analyze → Score → Alert → Trade → Monitor
        </p>
      </div>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {(kpis ?? []).map((k) => (<KpiCard key={k.id} kpi={k} />))}
      </div>
      <div className="grid grid-cols-1 gap-5 xl:grid-cols-3">
        <div className="xl:col-span-2"><LiveTokenFeed /></div>
        <div className="min-h-[420px]"><AlertsTimeline /></div>
      </div>
      <div className="min-h-[340px]"><OpportunityRadar /></div>
    </div>
  );
}
```

- [ ] **Step 5: `app/(app)/page.tsx`** — RSC prefetch + HydrationBoundary

```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { OverviewContent } from "@/components/dashboard/OverviewContent";

export default async function OverviewPage() {
  const queryClient = getQueryClient();
  const api = getApi();
  await Promise.all([
    queryClient.prefetchQuery({ queryKey: qk.kpis, queryFn: () => api.getKpis() }),
    queryClient.prefetchQuery({ queryKey: qk.tokens, queryFn: () => api.getTokens() }),
    queryClient.prefetchQuery({ queryKey: qk.alerts, queryFn: () => api.getAlerts() }),
    queryClient.prefetchQuery({ queryKey: qk.radar, queryFn: () => api.getRadar() }),
  ]);
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <OverviewContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 6: `LiveTokenFeed.test.tsx`** (sort + fresh highlight)

```tsx
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient } from "@/lib/get-query-client";
import { LiveTokenFeed } from "./LiveTokenFeed";

function wrap() {
  const client = getQueryClient();
  return render(<QueryClientProvider client={client}><LiveTokenFeed /></QueryClientProvider>);
}

test("renders rows and marks <60s tokens fresh", async () => {
  wrap();
  await waitFor(() => expect(screen.getByText("SolPulse")).toBeInTheDocument());
  const gfrogRow = screen.getByText("GigaFrog").closest("tr")!;
  expect(gfrogRow.getAttribute("data-fresh")).toBe("true"); // 12s old
});

test("sorting by liquidity puts highest first", async () => {
  wrap();
  await waitFor(() => expect(screen.getByText("SolPulse")).toBeInTheDocument());
  await userEvent.click(screen.getByRole("button", { name: /Liq/ }));
  const firstRow = screen.getAllByRole("row")[1];
  expect(within(firstRow).getByText("Helios")).toBeInTheDocument(); // $320K liquidity
});
```

- [ ] **Step 7: Testleri ve build'i doğrula**

Run: `npm run test -- dashboard`
Expected: PASS.
Run: `npm run build`
Expected: derleme başarılı.

- [ ] **Step 8: Commit**

```bash
git add apps/web/components/dashboard "apps/web/app/(app)/page.tsx"
git commit -m "feat(web): build Overview dashboard with RSC prefetch and hydration"
```

---

### Task 10: Real-time mock stream + kabul kontrolü

**Files:**
- Create: `apps/web/lib/hooks/live.ts`
- Modify: `apps/web/components/dashboard/OverviewContent.tsx` (canlı abonelik bağlama)
- Test: `apps/web/lib/hooks/live.test.tsx`

**Interfaces:**
- Consumes: `getApi()` (subscribe*), `getQueryClient/qk`, `useQueryClient`.
- Produces: `useLiveTokens()`, `useLiveAlerts()` — abonelikle query cache'i patch'ler.

- [ ] **Step 1: `lib/hooks/live.ts`** (WS → cache patch deseni)

```ts
"use client";
import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getApi } from "@/lib/api";
import { qk } from "@/lib/get-query-client";
import type { TokenRow, AlertEvent } from "@/lib/api/types";

export function useLiveTokens() {
  const qc = useQueryClient();
  useEffect(() => getApi().subscribeTokens((tokens: TokenRow[]) => {
    qc.setQueryData(qk.tokens, tokens);
  }), [qc]);
}

export function useLiveAlerts() {
  const qc = useQueryClient();
  useEffect(() => getApi().subscribeAlerts((alert: AlertEvent) => {
    qc.setQueryData<AlertEvent[]>(qk.alerts, (prev) => [alert, ...(prev ?? [])].slice(0, 20));
  }), [qc]);
}
```

- [ ] **Step 2: `OverviewContent.tsx`'e abonelikleri bağla** — bileşenin başına ekle:

```tsx
import { useLiveTokens, useLiveAlerts } from "@/lib/hooks/live";
// ... fonksiyon gövdesinin başında:
useLiveTokens();
useLiveAlerts();
```

- [ ] **Step 3: `live.test.tsx`** (abonelik cache'i güncelliyor + unsubscribe)

```tsx
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { useLiveAlerts } from "./live";

function Probe() {
  useLiveAlerts();
  return <div>probe</div>;
}

test("useLiveAlerts pushes new alerts into cache", async () => {
  const client = getQueryClient();
  client.setQueryData?.(qk.alerts, []);
  render(<QueryClientProvider client={client}><Probe /></QueryClientProvider>);
  await waitFor(() => {
    const alerts = client.getQueryData(qk.alerts) as unknown[] | undefined;
    expect((alerts?.length ?? 0)).toBeGreaterThan(0);
  }, { timeout: 7000 });
});
```

Not: mock `subscribeAlerts` 5s aralıkla yayar; test timeout 7s.

- [ ] **Step 4: Testi doğrula**

Run: `npm run test -- hooks/live`
Expected: PASS.

- [ ] **Step 5: Tam kabul kontrolü**

Run: `npm run test`
Expected: tüm testler PASS.
Run: `npm run build && npm run dev`
Manuel: `/` açılır → dark tema, 8 KPI, LiveTokenFeed (sort + watchlist + `<60s` yeşil highlight), OpportunityRadar, AlertsTimeline; sidebar collapse; trading mode Live seçilince header'da kırmızı şerit; birkaç saniye içinde canlı bir güncelleme görülür; `/tokens` vb. placeholder açılır.

- [ ] **Step 6: Commit**

```bash
git add apps/web/lib/hooks/live.ts apps/web/components/dashboard/OverviewContent.tsx
git commit -m "feat(web): wire mock real-time stream into query cache"
```

---

## Self-Review (yazar kontrolü)

**Spec coverage:**
- Next.js server-first + client island'lar → Task 1 (scaffold, layout), Task 9 (RSC prefetch). ✅
- Tailwind v4 + Sentinel token'ları → Task 1 Step 5. ✅
- shadcn/ui → Task 3. ✅
- Veri seam (contract/mock/http/getApi, hiçbir bileşen mock import etmez) → Task 4; bileşenler hook kullanıyor (Task 7/9). ✅
- TanStack Query + hooks + Zustand → Task 5, 6. ✅
- Design system dark-only, skor seviyeleri → Task 1, Task 2. ✅
- App shell (Sidebar/Header/TradingModeBadge, RPC/Solana/Telegram footer, Emergency Pause, Live indicator) → Task 8. ✅
- 17 route (Overview gerçek + 16 placeholder) → Task 8. ✅
- Overview: 8 KPI, LiveTokenFeed (sort/watchlist/highlight), OpportunityRadar, AlertsTimeline → Task 9. ✅
- Sentinel primitives (ScoreBadge, Sparkline, TokenAvatar, WalletAddress copy-toast, KpiCard) → Task 7. ✅
- Real-time mock stream (subscribe → cache) → Task 10. ✅
- Test stratejisi (format, scoreToLevel, adapter, ScoreBadge/WalletAddress/LiveTokenFeed, smoke) → Task 1,2,4,7,8,9,10. ✅
- `.env.example` (DATA_SOURCE/API_BASE_URL/WS_URL) → Task 1 Step 8. ✅

**Kapsam dışı doğrulama:** lightweight-charts, Cytoscape, light tema, mobil, auth, command palette, geniş shadcn seti — hiçbiri task'lara girmedi (bilinçli). ✅

**Placeholder taraması:** kod adımlarında gerçek kod var; "TODO(backend)" yalnız `http.ts` stub'ında ve bilinçli. ✅

**Tip tutarlılığı:** `SentinelApi` imzaları Task 4'te tanımlandığı gibi Task 5/10'da kullanıldı; `qk` anahtarları tutarlı; `TradingMode` tek kaynaktan (`lib/store/session`); `Kpi/TokenRow/AlertEvent/RadarPoint` `lib/api/types`'ten. ✅
