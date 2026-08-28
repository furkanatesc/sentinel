# Research Assistant (Ekran 11) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Research Assistant screen (`/research`): a multi-turn chat panel that streams sourced, "informational analysis"-labeled answers over the existing mock seam.

**Architecture:** Components read a `useResearch()` hook; the hook calls `getApi()` and drives a pure Zustand store (store imports no mock/http → DIP). Streaming reuses the existing `subscribe*(cb) => unsubscribe` seam shape: `streamResearchAnswer` pushes word chunks then a final answer with sources. A pure `pickAnswer` router maps free-text questions to canned answers; `SOURCE_KIND_DEFS` maps source kinds to labels/icons/hrefs (OCP).

**Tech Stack:** Next.js 16 App Router, TypeScript, Tailwind v4, shadcn/Base-UI (`Button`, `Badge`, `ScrollArea`), Zustand, TanStack Query (suggestions only), Vitest + React Testing Library. Dark-only Turkish UI.

**Spec:** `docs/superpowers/specs/2026-08-28-sentinel-research-assistant-design.md`

## Global Constraints

- Turkish user-facing copy; dark-only theme; follow existing Sentinel design tokens + `cn` util.
- **DIP:** no component imports `mock`/`http` directly; components → `useResearch()`/`useQuery(getApi())`, hook → `getApi()`. The Zustand store is pure state+reducers (no I/O import).
- Seam methods return through `getApi()` only. `streamResearchAnswer` mirrors `subscribe*`: returns a synchronous cancel `() => void`.
- `http.ts` stays not-live for both new methods; do **not** add them to `LIVE_ENDPOINTS`.
- TDD: failing test first, minimal impl, green, commit. Keep files focused (one responsibility).
- Existing suite must stay green; `npm run build` must succeed with `/research` as a working route.
- Test runner: `npx vitest run <path>` (jsdom env already configured in `vitest.config.ts`).

---

### Task 1: Seam foundation — types + contract + http

**Files:**
- Modify: `apps/web/lib/api/types.ts` (append Research types)
- Modify: `apps/web/lib/api/contract.ts` (add 2 methods to `SentinelApi`)
- Modify: `apps/web/lib/api/http.ts` (add 2 not-ready stubs)
- Test: `apps/web/lib/api/research.contract.test.ts` (Create)

**Interfaces:**
- Produces: `ResearchSourceKind`, `ResearchSource`, `ResearchAnswer`, `ResearchSuggestion` types; `SentinelApi.getResearchSuggestions(): Promise<ResearchSuggestion[]>`; `SentinelApi.streamResearchAnswer(question, onChunk, onDone): () => void`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/api/research.contract.test.ts`:
```ts
import { describe, it, expect, vi } from "vitest";
import { httpApi } from "./http";

describe("research seam — http not ready", () => {
  it("getResearchSuggestions rejects (backend not connected)", async () => {
    await expect(httpApi.getResearchSuggestions()).rejects.toThrow(/not implemented/i);
  });

  it("streamResearchAnswer throws synchronously (backend not connected)", () => {
    expect(() => httpApi.streamResearchAnswer("q", vi.fn(), vi.fn())).toThrow(/not implemented/i);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run lib/api/research.contract.test.ts`
Expected: FAIL — `getResearchSuggestions`/`streamResearchAnswer` do not exist on `httpApi` (type + runtime error).

- [ ] **Step 3: Add types to `types.ts`**

Append to `apps/web/lib/api/types.ts`:
```ts
export type ResearchSourceKind =
  | "token" | "creator" | "wallet" | "tx" | "risk-rule" | "strategy" | "timestamp";

export interface ResearchSource {
  id: string;
  kind: ResearchSourceKind;
  label: string;   // display label, e.g. "AERO", "6Rt4…9kQ", "rug-pull-rule"
  ref?: string;    // target id (mint | address | tx sig | strategy id); href built in presentation
}

export interface ResearchAnswer {
  text: string;
  sources: ResearchSource[];
}

export interface ResearchSuggestion {
  id: string;
  text: string;
}
```

- [ ] **Step 4: Add methods to `contract.ts`**

In `apps/web/lib/api/contract.ts`, add the new type names to the top `import type { ... }` list (`ResearchSuggestion, ResearchAnswer, ResearchSource`), then add to the `SentinelApi` interface (before the real-time seam comment block):
```ts
  getResearchSuggestions(): Promise<ResearchSuggestion[]>;
  /** Streaming seam — subscribe pattern. onChunk = incremental text, onDone = final answer + sources. Returns cancel fn. */
  streamResearchAnswer(
    question: string,
    onChunk: (chunk: string) => void,
    onDone: (answer: ResearchAnswer) => void,
  ): () => void;
```

- [ ] **Step 5: Add not-ready stubs to `http.ts`**

In `apps/web/lib/api/http.ts`, inside the `httpApi` object (near `getSystemHealth`), add:
```ts
  getResearchSuggestions: notReady,
  streamResearchAnswer: () => { throw new Error("httpApi not implemented — backend not connected yet"); },
```
(`notReady` is the existing `() => Promise.reject(...)`; the stream method must throw synchronously because it returns a cancel fn, not a Promise.)

- [ ] **Step 6: Run test to verify it passes**

Run: `npx vitest run lib/api/research.contract.test.ts`
Expected: PASS.

- [ ] **Step 7: Typecheck the seam**

Run: `npx tsc --noEmit`
Expected: no errors (mock still satisfies `SentinelApi`? — NOT yet; mock impl lands in Task 4). If `mock.ts` now fails to satisfy `SentinelApi`, that is expected — proceed; it is fixed in Task 4. To keep this task green in isolation, temporarily nothing is needed since the contract test only touches `httpApi`. Do **not** edit `mock.ts` here.

> Note: `npx tsc --noEmit` may report `mockApi` missing the 2 methods until Task 4. That is an accepted interim state between Task 1 and Task 4; the per-task vitest run is the gate for this task.

- [ ] **Step 8: Commit**

```bash
git add apps/web/lib/api/types.ts apps/web/lib/api/contract.ts apps/web/lib/api/http.ts apps/web/lib/api/research.contract.test.ts
git commit -m "feat(research): seam foundation — types + contract + http stubs"
```

---

### Task 2: Source-kind registry (`SOURCE_KIND_DEFS`)

**Files:**
- Create: `apps/web/lib/research/source-defs.ts`
- Test: `apps/web/lib/research/source-defs.test.ts`

**Interfaces:**
- Consumes: `ResearchSource`, `ResearchSourceKind` (Task 1).
- Produces: `SOURCE_KIND_DEFS: Record<ResearchSourceKind, SourceKindDef>`; `SourceKindDef = { label: string; icon: LucideIcon; buildHref?: (src: ResearchSource) => string }`; `hrefForSource(src): string | undefined`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/research/source-defs.test.ts`:
```ts
import { describe, it, expect } from "vitest";
import { hrefForSource, SOURCE_KIND_DEFS } from "./source-defs";

describe("SOURCE_KIND_DEFS / hrefForSource", () => {
  it("token → /tokens/[ref]", () => {
    expect(hrefForSource({ id: "s1", kind: "token", label: "AERO", ref: "AERO" })).toBe("/tokens/AERO");
  });
  it("creator → /creators/[ref]", () => {
    expect(hrefForSource({ id: "s2", kind: "creator", label: "6Rt4", ref: "6Rt4abc" })).toBe("/creators/6Rt4abc");
  });
  it("wallet → /wallet-graph", () => {
    expect(hrefForSource({ id: "s3", kind: "wallet", label: "9kQ", ref: "9kQxyz" })).toBe("/wallet-graph");
  });
  it("tx / risk-rule / timestamp have no href", () => {
    expect(hrefForSource({ id: "s4", kind: "tx", label: "5xA…" })).toBeUndefined();
    expect(hrefForSource({ id: "s5", kind: "risk-rule", label: "rug-pull-rule" })).toBeUndefined();
    expect(hrefForSource({ id: "s6", kind: "timestamp", label: "12:04" })).toBeUndefined();
  });
  it("every kind has a def with label + icon", () => {
    (["token","creator","wallet","tx","risk-rule","strategy","timestamp"] as const).forEach((k) => {
      expect(SOURCE_KIND_DEFS[k].label).toBeTruthy();
      expect(SOURCE_KIND_DEFS[k].icon).toBeTruthy();
    });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run lib/research/source-defs.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the registry**

Create `apps/web/lib/research/source-defs.ts`:
```ts
import { Coins, UserSearch, Share2, Hash, ShieldAlert, Layers, Clock, type LucideIcon } from "lucide-react";
import type { ResearchSource, ResearchSourceKind } from "@/lib/api/types";

export interface SourceKindDef {
  label: string;                                   // human label for the kind
  icon: LucideIcon;
  buildHref?: (src: ResearchSource) => string;     // absent → non-clickable chip
}

export const SOURCE_KIND_DEFS: Record<ResearchSourceKind, SourceKindDef> = {
  token:       { label: "Token",     icon: Coins,      buildHref: (s) => `/tokens/${s.ref}` },
  creator:     { label: "Üretici",   icon: UserSearch, buildHref: (s) => `/creators/${s.ref}` },
  wallet:      { label: "Cüzdan",    icon: Share2,     buildHref: () => `/wallet-graph` },
  tx:          { label: "İşlem",     icon: Hash },
  "risk-rule": { label: "Risk Kuralı", icon: ShieldAlert },
  strategy:    { label: "Strateji",  icon: Layers },   // linksiz (spec default); ileride /strategies/[ref]
  timestamp:   { label: "Zaman",     icon: Clock },
};

export function hrefForSource(src: ResearchSource): string | undefined {
  const def = SOURCE_KIND_DEFS[src.kind];
  return def.buildHref && src.ref !== undefined ? def.buildHref(src) : def.buildHref?.(src);
}
```
> `wallet`'s `buildHref` ignores `ref` (route has no param yet), so it must return even when `ref` is set; `hrefForSource` calls `buildHref` when present (the `def.buildHref?.(src)` fallback covers the ref-less wallet case). token/creator require `ref`.

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run lib/research/source-defs.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/research/source-defs.ts apps/web/lib/research/source-defs.test.ts
git commit -m "feat(research): SOURCE_KIND_DEFS registry + hrefForSource"
```

---

### Task 3: Answer router + canned data (`pickAnswer`, suggestions)

**Files:**
- Create: `apps/web/lib/research/match.ts`
- Test: `apps/web/lib/research/match.test.ts`

**Interfaces:**
- Consumes: `ResearchSource`, `ResearchSuggestion` (Task 1).
- Produces: `CannedAnswer = { text: string; sources: ResearchSource[] }`; `pickAnswer(question: string): CannedAnswer`; `RESEARCH_SUGGESTIONS: ResearchSuggestion[]`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/research/match.test.ts`:
```ts
import { describe, it, expect } from "vitest";
import { pickAnswer, RESEARCH_SUGGESTIONS } from "./match";

describe("pickAnswer", () => {
  it("routes risk questions to a risk answer with sources", () => {
    const a = pickAnswer("Bu token neden riskli?");
    expect(a.text.toLowerCase()).toContain("risk");
    expect(a.sources.length).toBeGreaterThan(0);
  });
  it("routes creator-history questions", () => {
    const a = pickAnswer("Üreticinin geçmişi nasıl?");
    expect(a.text).toBeTruthy();
    expect(a.sources.some((s) => s.kind === "creator")).toBe(true);
  });
  it("routes wallet-cluster questions", () => {
    const a = pickAnswer("Cüzdan kümesi şüpheli mi?");
    expect(a.sources.some((s) => s.kind === "wallet")).toBe(true);
  });
  it("falls back for unknown questions", () => {
    const a = pickAnswer("qwerty zxcv unknown");
    expect(a.text).toBeTruthy();
    expect(a.sources).toEqual([]);
  });
  it("is case/accent tolerant (matches on lowercase keywords)", () => {
    expect(pickAnswer("NEDEN RİSKLİ").text).toBe(pickAnswer("neden riskli").text);
  });
});

describe("RESEARCH_SUGGESTIONS", () => {
  it("has several starter questions with unique ids", () => {
    expect(RESEARCH_SUGGESTIONS.length).toBeGreaterThanOrEqual(4);
    expect(new Set(RESEARCH_SUGGESTIONS.map((s) => s.id)).size).toBe(RESEARCH_SUGGESTIONS.length);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run lib/research/match.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the router + data**

Create `apps/web/lib/research/match.ts`:
```ts
import type { ResearchSource, ResearchSuggestion } from "@/lib/api/types";

export interface CannedAnswer { text: string; sources: ResearchSource[]; }

interface Rule { keywords: string[]; answer: CannedAnswer; }

// mock token/wallet refs kept consistent with mock.ts data (GFROG rug, LMN whale, 6Rt4 wallet)
const RULES: Rule[] = [
  {
    keywords: ["riskli", "risk", "neden riskli", "tehlike"],
    answer: {
      text: "GFROG yüksek riskli görünüyor: üretici likidite havuzunun %92'sini çekti ve ilk 5 cüzdan arzın %78'ini elinde tutuyor. Bu, rug-pull kalıbıyla eşleşiyor; güvenlik skoru 22/100.",
      sources: [
        { id: "r1", kind: "token", label: "GFROG", ref: "GFROG" },
        { id: "r2", kind: "risk-rule", label: "likidite-cekildi" },
        { id: "r3", kind: "wallet", label: "7mLp…1Qw8", ref: "7mLp2c1Qw8" },
      ],
    },
  },
  {
    keywords: ["üretici", "uretici", "creator", "geçmiş", "gecmis"],
    answer: {
      text: "Bu üreticinin 6 önceki tokenından 4'ü rug ile sonuçlandı, 1'i graduate oldu. Ortalama likidite ömrü 3 saat. Üretici itibar skoru 29/100.",
      sources: [
        { id: "c1", kind: "creator", label: "6Rt4…9kQ", ref: "6Rt4abc9kQ" },
        { id: "c2", kind: "token", label: "ZAP", ref: "ZAP" },
        { id: "c3", kind: "timestamp", label: "3 saat" },
      ],
    },
  },
  {
    keywords: ["küme", "kume", "cluster", "cüzdan", "cuzdan", "wallet"],
    answer: {
      text: "Wallet grafiği, 5 cüzdanın aynı funder tarafından beslendiğini ve GFROG'da eşzamanlı alım yaptığını gösteriyor. Bu koordineli davranış manipülasyon işareti.",
      sources: [
        { id: "w1", kind: "wallet", label: "6Rt4…9kQ", ref: "6Rt4abc9kQ" },
        { id: "w2", kind: "tx", label: "5xAr…t9Kp" },
        { id: "w3", kind: "token", label: "GFROG", ref: "GFROG" },
      ],
    },
  },
  {
    keywords: ["benzer", "performans", "benzer skor"],
    answer: {
      text: "Benzer güven skoruna (80+) sahip son 20 tokenın %65'i ilk saatte pozitif getiri sağladı; medyan tepe getiri +48%. HLS ve PULSE bu kohortta.",
      sources: [
        { id: "p1", kind: "token", label: "HLS", ref: "HLS" },
        { id: "p2", kind: "token", label: "PULSE", ref: "PULSE" },
      ],
    },
  },
  {
    keywords: ["sinyal", "neden üretildi", "neden uretildi", "signal"],
    answer: {
      text: "Sinyal, momentum 88'e çıkıp güvenlik skoru 78'in üstünde kalınca 'buy-momentum-v2' stratejisi tarafından üretildi. Tetikleyici: 5dk hacim > $40K.",
      sources: [
        { id: "s1", kind: "strategy", label: "buy-momentum-v2", ref: "buy-momentum-v2" },
        { id: "s2", kind: "token", label: "PULSE", ref: "PULSE" },
        { id: "s3", kind: "timestamp", label: "az önce" },
      ],
    },
  },
];

const FALLBACK: CannedAnswer = {
  text: "Bu soruya net bir analiz üretmek için yeterli sinyal bulamadım. Belirli bir token, üretici ya da cüzdan hakkında sorarsan (ör. 'GFROG neden riskli?') kaynaklı bir analiz sunabilirim.",
  sources: [],
};

export function pickAnswer(question: string): CannedAnswer {
  const q = question.toLocaleLowerCase("tr");
  const hit = RULES.find((r) => r.keywords.some((k) => q.includes(k)));
  return hit ? hit.answer : FALLBACK;
}

export const RESEARCH_SUGGESTIONS: ResearchSuggestion[] = [
  { id: "q1", text: "GFROG neden riskli?" },
  { id: "q2", text: "Bu üreticinin geçmişi nasıl?" },
  { id: "q3", text: "Cüzdan kümesi şüpheli mi?" },
  { id: "q4", text: "Benzer skorlu tokenların performansı ne?" },
  { id: "q5", text: "Bu sinyal neden üretildi?" },
];
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run lib/research/match.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/research/match.ts apps/web/lib/research/match.test.ts
git commit -m "feat(research): pickAnswer router + canned answers + suggestions"
```

---

### Task 4: Mock seam implementation

**Files:**
- Modify: `apps/web/lib/api/mock.ts` (add 2 methods + imports)
- Test: `apps/web/lib/api/research.mock.test.ts` (Create)

**Interfaces:**
- Consumes: `pickAnswer`, `RESEARCH_SUGGESTIONS` (Task 3); `mockApi` (existing).
- Produces: `mockApi.getResearchSuggestions`, `mockApi.streamResearchAnswer` (satisfies `SentinelApi`).

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/api/research.mock.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { mockApi } from "./mock";

describe("mockApi.getResearchSuggestions", () => {
  it("returns the suggestion list", async () => {
    const list = await mockApi.getResearchSuggestions();
    expect(list.length).toBeGreaterThanOrEqual(4);
    expect(list[0]).toHaveProperty("text");
  });
});

describe("mockApi.streamResearchAnswer", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("streams chunks then calls onDone with full text + sources", () => {
    const onChunk = vi.fn();
    const onDone = vi.fn();
    mockApi.streamResearchAnswer("GFROG neden riskli?", onChunk, onDone);
    vi.runAllTimers();
    expect(onChunk.mock.calls.length).toBeGreaterThan(1);
    expect(onDone).toHaveBeenCalledTimes(1);
    const answer = onDone.mock.calls[0][0];
    const streamed = onChunk.mock.calls.map((c) => c[0]).join("");
    expect(streamed).toBe(answer.text);
    expect(answer.sources.length).toBeGreaterThan(0);
  });

  it("cancel fn stops streaming (no onDone)", () => {
    const onChunk = vi.fn();
    const onDone = vi.fn();
    const cancel = mockApi.streamResearchAnswer("GFROG neden riskli?", onChunk, onDone);
    vi.advanceTimersByTime(35); // one chunk
    cancel();
    vi.runAllTimers();
    expect(onDone).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run lib/api/research.mock.test.ts`
Expected: FAIL — methods do not exist on `mockApi`.

- [ ] **Step 3: Implement in `mock.ts`**

At the top of `apps/web/lib/api/mock.ts`, add import:
```ts
import { pickAnswer, RESEARCH_SUGGESTIONS } from "@/lib/research/match";
```
Inside the `mockApi` object (near `getSystemHealth`, before the `subscribe*` block), add:
```ts
  getResearchSuggestions() {
    return delay(RESEARCH_SUGGESTIONS);
  },

  streamResearchAnswer(question, onChunk, onDone) {
    const answer = pickAnswer(question);
    const words = answer.text.split(" ");
    let i = 0;
    const id = setInterval(() => {
      if (i < words.length) {
        onChunk((i === 0 ? "" : " ") + words[i]);
        i++;
      } else {
        clearInterval(id);
        onDone({ text: answer.text, sources: answer.sources });
      }
    }, 35);
    return () => clearInterval(id);
  },
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run lib/api/research.mock.test.ts`
Expected: PASS.

- [ ] **Step 5: Verify seam type completeness**

Run: `npx tsc --noEmit`
Expected: no errors — `mockApi` now satisfies `SentinelApi` (Task 1 interim state resolved).

- [ ] **Step 6: Commit**

```bash
git add apps/web/lib/api/mock.ts apps/web/lib/api/research.mock.test.ts
git commit -m "feat(research): mock getResearchSuggestions + streamResearchAnswer"
```

---

### Task 5: Zustand store (pure state + reducers)

**Files:**
- Create: `apps/web/lib/store/research.ts`
- Test: `apps/web/lib/store/research.test.ts`

**Interfaces:**
- Consumes: `ResearchSource` (Task 1).
- Produces: `ChatMessage` type; `useResearchStore` with state `{ messages: ChatMessage[]; isStreaming: boolean }` and actions `addUserMessage(text): string`, `appendChunk(id, chunk): void`, `finishAnswer(id, sources): void`, `reset(): void`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/lib/store/research.test.ts`:
```ts
import { describe, it, expect, beforeEach } from "vitest";
import { useResearchStore } from "./research";

const reset = () => useResearchStore.getState().reset();

describe("research store", () => {
  beforeEach(reset);

  it("addUserMessage adds user + streaming assistant placeholder", () => {
    const id = useResearchStore.getState().addUserMessage("merhaba");
    const { messages, isStreaming } = useResearchStore.getState();
    expect(messages).toHaveLength(2);
    expect(messages[0]).toMatchObject({ role: "user", text: "merhaba" });
    expect(messages[1]).toMatchObject({ role: "assistant", text: "", status: "streaming", id });
    expect(isStreaming).toBe(true);
  });

  it("appendChunk accumulates assistant text", () => {
    const id = useResearchStore.getState().addUserMessage("q");
    useResearchStore.getState().appendChunk(id, "Merhaba");
    useResearchStore.getState().appendChunk(id, " dünya");
    expect(useResearchStore.getState().messages[1].text).toBe("Merhaba dünya");
  });

  it("finishAnswer attaches sources, marks done, clears isStreaming", () => {
    const id = useResearchStore.getState().addUserMessage("q");
    useResearchStore.getState().appendChunk(id, "cevap");
    useResearchStore.getState().finishAnswer(id, [{ id: "s1", kind: "token", label: "AERO", ref: "AERO" }]);
    const msg = useResearchStore.getState().messages[1];
    expect(msg.status).toBe("done");
    expect(msg.sources).toHaveLength(1);
    expect(useResearchStore.getState().isStreaming).toBe(false);
  });

  it("reset clears everything", () => {
    useResearchStore.getState().addUserMessage("q");
    reset();
    expect(useResearchStore.getState().messages).toEqual([]);
    expect(useResearchStore.getState().isStreaming).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run lib/store/research.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the store**

Create `apps/web/lib/store/research.ts`:
```ts
import { create } from "zustand";
import type { ResearchSource } from "@/lib/api/types";

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  text: string;
  sources?: ResearchSource[];
  status: "streaming" | "done";
}

interface ResearchState {
  messages: ChatMessage[];
  isStreaming: boolean;
  addUserMessage: (text: string) => string;
  appendChunk: (assistantId: string, chunk: string) => void;
  finishAnswer: (assistantId: string, sources: ResearchSource[]) => void;
  reset: () => void;
}

let seq = 0;
const nextId = () => `m${Date.now()}-${seq++}`;

export const useResearchStore = create<ResearchState>((set) => ({
  messages: [],
  isStreaming: false,
  addUserMessage: (text) => {
    const assistantId = nextId();
    set((s) => ({
      isStreaming: true,
      messages: [
        ...s.messages,
        { id: nextId(), role: "user", text, status: "done" },
        { id: assistantId, role: "assistant", text: "", status: "streaming" },
      ],
    }));
    return assistantId;
  },
  appendChunk: (assistantId, chunk) =>
    set((s) => ({
      messages: s.messages.map((m) => (m.id === assistantId ? { ...m, text: m.text + chunk } : m)),
    })),
  finishAnswer: (assistantId, sources) =>
    set((s) => ({
      isStreaming: false,
      messages: s.messages.map((m) =>
        m.id === assistantId ? { ...m, sources, status: "done" as const } : m,
      ),
    })),
  reset: () => set({ messages: [], isStreaming: false }),
}));
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run lib/store/research.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/lib/store/research.ts apps/web/lib/store/research.test.ts
git commit -m "feat(research): pure Zustand chat store"
```

---

### Task 6: `useResearch` orchestration hook

**Files:**
- Create: `apps/web/components/research/use-research.ts`
- Test: `apps/web/components/research/use-research.test.ts`

**Interfaces:**
- Consumes: `getApi()` (existing), `useResearchStore` (Task 5).
- Produces: `useResearch(): { messages: ChatMessage[]; isStreaming: boolean; send: (q: string) => void; stop: () => void }`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/research/use-research.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useResearch } from "./use-research";
import { useResearchStore } from "@/lib/store/research";

beforeEach(() => { useResearchStore.getState().reset(); vi.useFakeTimers(); });
afterEach(() => vi.useRealTimers());

describe("useResearch", () => {
  it("send streams an answer into the store", () => {
    const { result } = renderHook(() => useResearch());
    act(() => result.current.send("GFROG neden riskli?"));
    expect(result.current.isStreaming).toBe(true);
    act(() => vi.runAllTimers());
    const last = result.current.messages.at(-1)!;
    expect(last.role).toBe("assistant");
    expect(last.status).toBe("done");
    expect(last.text.length).toBeGreaterThan(0);
    expect(result.current.isStreaming).toBe(false);
  });

  it("send is ignored while already streaming", () => {
    const { result } = renderHook(() => useResearch());
    act(() => result.current.send("GFROG neden riskli?"));
    act(() => result.current.send("ikinci soru")); // should be ignored
    const userMsgs = result.current.messages.filter((m) => m.role === "user");
    expect(userMsgs).toHaveLength(1);
  });

  it("stop halts streaming and marks the answer done", () => {
    const { result } = renderHook(() => useResearch());
    act(() => result.current.send("GFROG neden riskli?"));
    act(() => vi.advanceTimersByTime(35));
    act(() => result.current.stop());
    expect(result.current.isStreaming).toBe(false);
    expect(result.current.messages.at(-1)!.status).toBe("done");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run components/research/use-research.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Write the hook**

Create `apps/web/components/research/use-research.ts`:
```ts
"use client";
import { useEffect, useRef } from "react";
import { getApi } from "@/lib/api";
import { useResearchStore } from "@/lib/store/research";

export function useResearch() {
  const messages = useResearchStore((s) => s.messages);
  const isStreaming = useResearchStore((s) => s.isStreaming);
  const cancelRef = useRef<(() => void) | null>(null);
  const activeIdRef = useRef<string | null>(null);

  const send = (question: string) => {
    const q = question.trim();
    if (!q || useResearchStore.getState().isStreaming) return;
    const store = useResearchStore.getState();
    const assistantId = store.addUserMessage(q);
    activeIdRef.current = assistantId;
    cancelRef.current = getApi().streamResearchAnswer(
      q,
      (chunk) => useResearchStore.getState().appendChunk(assistantId, chunk),
      (answer) => {
        useResearchStore.getState().finishAnswer(assistantId, answer.sources);
        cancelRef.current = null;
        activeIdRef.current = null;
      },
    );
  };

  const stop = () => {
    cancelRef.current?.();
    cancelRef.current = null;
    const id = activeIdRef.current;
    if (id) useResearchStore.getState().finishAnswer(id, []);
    activeIdRef.current = null;
  };

  useEffect(() => () => { cancelRef.current?.(); }, []); // cancel on unmount

  return { messages, isStreaming, send, stop };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run components/research/use-research.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/research/use-research.ts apps/web/components/research/use-research.test.ts
git commit -m "feat(research): useResearch orchestration hook"
```

---

### Task 7: Leaf presentational components

**Files:**
- Create: `apps/web/components/research/InfoDisclaimerBanner.tsx`
- Create: `apps/web/components/research/SourceChip.tsx`
- Create: `apps/web/components/research/SourceList.tsx`
- Create: `apps/web/components/research/SuggestionChips.tsx`
- Create: `apps/web/components/research/ChatMessageBubble.tsx`
- Test: `apps/web/components/research/research-ui.test.tsx`

**Interfaces:**
- Consumes: `ChatMessage` (Task 5), `ResearchSource`/`ResearchSuggestion` (Task 1), `SOURCE_KIND_DEFS`/`hrefForSource` (Task 2).
- Produces: `InfoDisclaimerBanner`, `SourceChip({ source })`, `SourceList({ sources })`, `SuggestionChips({ suggestions, onPick, disabled })`, `ChatMessageBubble({ message })`.

- [ ] **Step 1: Write the failing test**

Create `apps/web/components/research/research-ui.test.tsx`:
```tsx
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SourceChip } from "./SourceChip";
import { SuggestionChips } from "./SuggestionChips";
import { ChatMessageBubble } from "./ChatMessageBubble";
import { InfoDisclaimerBanner } from "./InfoDisclaimerBanner";

describe("SourceChip", () => {
  it("renders a link for token sources", () => {
    render(<SourceChip source={{ id: "s1", kind: "token", label: "AERO", ref: "AERO" }} />);
    expect(screen.getByRole("link", { name: /AERO/ })).toHaveAttribute("href", "/tokens/AERO");
  });
  it("renders a non-link chip for tx sources", () => {
    render(<SourceChip source={{ id: "s2", kind: "tx", label: "5xA…" }} />);
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText(/5xA/)).toBeInTheDocument();
  });
});

describe("SuggestionChips", () => {
  it("fires onPick with the suggestion text", () => {
    const onPick = vi.fn();
    render(<SuggestionChips suggestions={[{ id: "q1", text: "GFROG neden riskli?" }]} onPick={onPick} disabled={false} />);
    fireEvent.click(screen.getByRole("button", { name: /GFROG/ }));
    expect(onPick).toHaveBeenCalledWith("GFROG neden riskli?");
  });
  it("disables chips while streaming", () => {
    render(<SuggestionChips suggestions={[{ id: "q1", text: "x" }]} onPick={() => {}} disabled />);
    expect(screen.getByRole("button", { name: "x" })).toBeDisabled();
  });
});

describe("ChatMessageBubble", () => {
  it("shows a streaming cursor while streaming", () => {
    render(<ChatMessageBubble message={{ id: "a1", role: "assistant", text: "yaz", status: "streaming" }} />);
    expect(screen.getByTestId("stream-cursor")).toBeInTheDocument();
  });
  it("renders sources when done", () => {
    render(<ChatMessageBubble message={{ id: "a2", role: "assistant", text: "bitti", status: "done",
      sources: [{ id: "s1", kind: "token", label: "AERO", ref: "AERO" }] }} />);
    expect(screen.getByRole("link", { name: /AERO/ })).toBeInTheDocument();
  });
});

describe("InfoDisclaimerBanner", () => {
  it("renders the informational-analysis label", () => {
    render(<InfoDisclaimerBanner />);
    expect(screen.getByText(/bilgilendirme/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run components/research/research-ui.test.tsx`
Expected: FAIL — modules not found.

- [ ] **Step 3: Write the components**

Create `apps/web/components/research/InfoDisclaimerBanner.tsx`:
```tsx
import { Info } from "lucide-react";

export function InfoDisclaimerBanner() {
  return (
    <div className="flex items-center gap-2 rounded-md border border-border/60 bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
      <Info className="h-3.5 w-3.5 shrink-0" />
      <span>Bu analizler yalnızca bilgilendirme amaçlıdır; trade kararı ya da yatırım tavsiyesi değildir.</span>
    </div>
  );
}
```

Create `apps/web/components/research/SourceChip.tsx`:
```tsx
import Link from "next/link";
import type { ResearchSource } from "@/lib/api/types";
import { SOURCE_KIND_DEFS, hrefForSource } from "@/lib/research/source-defs";
import { cn } from "@/lib/utils";

export function SourceChip({ source }: { source: ResearchSource }) {
  const def = SOURCE_KIND_DEFS[source.kind];
  const Icon = def.icon;
  const href = hrefForSource(source);
  const cls = cn(
    "inline-flex items-center gap-1 rounded-full border border-border/60 bg-card px-2 py-0.5 text-[11px] text-muted-foreground",
    href && "hover:border-primary/60 hover:text-foreground transition-colors",
  );
  const inner = (
    <>
      <Icon className="h-3 w-3" />
      <span className="opacity-70">{def.label}:</span>
      <span className="font-medium text-foreground/90">{source.label}</span>
    </>
  );
  return href ? <Link href={href} className={cls}>{inner}</Link> : <span className={cls}>{inner}</span>;
}
```

Create `apps/web/components/research/SourceList.tsx`:
```tsx
import type { ResearchSource } from "@/lib/api/types";
import { SourceChip } from "./SourceChip";

export function SourceList({ sources }: { sources: ResearchSource[] }) {
  if (!sources.length) return null;
  return (
    <div className="mt-2 flex flex-wrap gap-1.5">
      <span className="text-[11px] text-muted-foreground/70">Kaynaklar:</span>
      {sources.map((s) => <SourceChip key={s.id} source={s} />)}
    </div>
  );
}
```

Create `apps/web/components/research/SuggestionChips.tsx`:
```tsx
import type { ResearchSuggestion } from "@/lib/api/types";
import { Button } from "@/components/ui/button";

export function SuggestionChips({
  suggestions, onPick, disabled,
}: { suggestions: ResearchSuggestion[]; onPick: (text: string) => void; disabled: boolean }) {
  return (
    <div className="flex flex-wrap gap-2">
      {suggestions.map((s) => (
        <Button key={s.id} variant="outline" size="sm" disabled={disabled}
          className="h-auto whitespace-normal py-1.5 text-left text-xs" onClick={() => onPick(s.text)}>
          {s.text}
        </Button>
      ))}
    </div>
  );
}
```

Create `apps/web/components/research/ChatMessageBubble.tsx`:
```tsx
import type { ChatMessage } from "@/lib/store/research";
import { SourceList } from "./SourceList";
import { cn } from "@/lib/utils";

export function ChatMessageBubble({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user";
  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div className={cn(
        "max-w-[80%] rounded-lg px-3 py-2 text-sm",
        isUser ? "bg-primary/15 text-foreground" : "bg-card border border-border/60",
      )}>
        <p className="whitespace-pre-wrap leading-relaxed">
          {message.text}
          {message.status === "streaming" && (
            <span data-testid="stream-cursor" className="ml-0.5 inline-block animate-pulse">▍</span>
          )}
        </p>
        {!isUser && message.status === "done" && message.sources && <SourceList sources={message.sources} />}
      </div>
    </div>
  );
}
```

> If `@/lib/utils` `cn` path differs, match the existing import used by `components/ui/button.tsx` (check its top import). Use the same source of `cn`.

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run components/research/research-ui.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/research/InfoDisclaimerBanner.tsx apps/web/components/research/SourceChip.tsx apps/web/components/research/SourceList.tsx apps/web/components/research/SuggestionChips.tsx apps/web/components/research/ChatMessageBubble.tsx apps/web/components/research/research-ui.test.tsx
git commit -m "feat(research): leaf presentational components"
```

---

### Task 8: ChatThread + ChatComposer + ResearchContent

**Files:**
- Create: `apps/web/components/research/ChatThread.tsx`
- Create: `apps/web/components/research/ChatComposer.tsx`
- Create: `apps/web/components/research/ResearchContent.tsx`
- Test: `apps/web/components/research/research-content.test.tsx`

**Interfaces:**
- Consumes: `useResearch` (Task 6), `ChatMessageBubble`/`SuggestionChips`/`InfoDisclaimerBanner` (Task 7), `getApi().getResearchSuggestions` (Task 4) via `useQuery`, `qk.researchSuggestions` (added here).
- Produces: `ResearchContent` (default-exported client screen); `ChatThread({ messages, suggestions, onPick, isStreaming })`; `ChatComposer({ onSend, onStop, isStreaming })`.

- [ ] **Step 1: Add the query key**

In `apps/web/lib/get-query-client.ts`, add to the `qk` object:
```ts
  researchSuggestions: ["research-suggestions"] as const,
```

- [ ] **Step 2: Write the failing test**

Create `apps/web/components/research/research-content.test.tsx`:
```tsx
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ResearchContent } from "./ResearchContent";
import { useResearchStore } from "@/lib/store/research";

function renderWithQuery(ui: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

beforeEach(() => useResearchStore.getState().reset());

describe("ResearchContent", () => {
  it("shows suggestions on empty thread and disclaimer", async () => {
    renderWithQuery(<ResearchContent />);
    expect(screen.getByText(/bilgilendirme/i)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: /GFROG neden riskli/ })).toBeInTheDocument());
  });

  it("sends a question via the composer and renders the streamed answer", async () => {
    renderWithQuery(<ResearchContent />);
    const box = screen.getByPlaceholderText(/soru/i);
    fireEvent.change(box, { target: { value: "GFROG neden riskli?" } });
    fireEvent.keyDown(box, { key: "Enter" });
    await waitFor(() => expect(screen.getByText(/GFROG neden riskli\?/)).toBeInTheDocument()); // user bubble
    await act(async () => { await new Promise((r) => setTimeout(r, 1500)); }); // let stream finish (real timers)
    await waitFor(() => expect(useResearchStore.getState().isStreaming).toBe(false));
    expect(useResearchStore.getState().messages.at(-1)!.status).toBe("done");
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `npx vitest run components/research/research-content.test.tsx`
Expected: FAIL — modules not found.

- [ ] **Step 4: Write `ChatComposer.tsx`**

Create `apps/web/components/research/ChatComposer.tsx`:
```tsx
"use client";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Send, Square } from "lucide-react";

export function ChatComposer({
  onSend, onStop, isStreaming,
}: { onSend: (q: string) => void; onStop: () => void; isStreaming: boolean }) {
  const [value, setValue] = useState("");

  const submit = () => {
    const q = value.trim();
    if (!q || isStreaming) return;
    onSend(q);
    setValue("");
  };

  return (
    <div className="flex items-end gap-2 border-t border-border/60 pt-3">
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit(); } }}
        disabled={isStreaming}
        placeholder="Bir soru sor (ör. GFROG neden riskli?)"
        rows={1}
        className="min-h-10 max-h-40 flex-1 resize-y rounded-md border border-border/60 bg-card px-3 py-2 text-sm outline-none focus:border-primary/60 disabled:opacity-50"
      />
      {isStreaming ? (
        <Button variant="outline" size="sm" onClick={onStop} className="gap-1">
          <Square className="h-3.5 w-3.5" /> Durdur
        </Button>
      ) : (
        <Button size="sm" onClick={submit} disabled={!value.trim()} className="gap-1">
          <Send className="h-3.5 w-3.5" /> Gönder
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 5: Write `ChatThread.tsx`**

Create `apps/web/components/research/ChatThread.tsx`:
```tsx
"use client";
import { useEffect, useRef } from "react";
import type { ChatMessage } from "@/lib/store/research";
import type { ResearchSuggestion } from "@/lib/api/types";
import { ChatMessageBubble } from "./ChatMessageBubble";
import { SuggestionChips } from "./SuggestionChips";

export function ChatThread({
  messages, suggestions, onPick, isStreaming,
}: {
  messages: ChatMessage[];
  suggestions: ResearchSuggestion[];
  onPick: (text: string) => void;
  isStreaming: boolean;
}) {
  const endRef = useRef<HTMLDivElement>(null);
  useEffect(() => { endRef.current?.scrollIntoView({ behavior: "smooth" }); }, [messages]);

  return (
    <div className="flex-1 space-y-3 overflow-y-auto pr-1">
      {messages.length === 0 ? (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Token, üretici ya da cüzdanlar hakkında kaynaklı bir analiz sorabilirsin. Örnek sorular:
          </p>
          <SuggestionChips suggestions={suggestions} onPick={onPick} disabled={isStreaming} />
        </div>
      ) : (
        messages.map((m) => <ChatMessageBubble key={m.id} message={m} />)
      )}
      <div ref={endRef} />
    </div>
  );
}
```

- [ ] **Step 6: Write `ResearchContent.tsx`**

Create `apps/web/components/research/ResearchContent.tsx`:
```tsx
"use client";
import { useQuery } from "@tanstack/react-query";
import { getApi } from "@/lib/api";
import { qk } from "@/lib/get-query-client";
import { useResearch } from "./use-research";
import { ChatThread } from "./ChatThread";
import { ChatComposer } from "./ChatComposer";
import { InfoDisclaimerBanner } from "./InfoDisclaimerBanner";

export function ResearchContent() {
  const { messages, isStreaming, send, stop } = useResearch();
  const { data: suggestions = [] } = useQuery({
    queryKey: qk.researchSuggestions,
    queryFn: () => getApi().getResearchSuggestions(),
  });

  return (
    <div className="mx-auto flex h-[calc(100vh-8rem)] max-w-3xl flex-col gap-3">
      <div>
        <h1 className="text-lg font-semibold">Araştırma Asistanı</h1>
        <p className="text-sm text-muted-foreground">AI destekli, kaynaklı token & cüzdan analizi.</p>
      </div>
      <InfoDisclaimerBanner />
      <ChatThread messages={messages} suggestions={suggestions} onPick={send} isStreaming={isStreaming} />
      <ChatComposer onSend={send} onStop={stop} isStreaming={isStreaming} />
    </div>
  );
}
```

- [ ] **Step 7: Run test to verify it passes**

Run: `npx vitest run components/research/research-content.test.tsx`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/web/lib/get-query-client.ts apps/web/components/research/ChatThread.tsx apps/web/components/research/ChatComposer.tsx apps/web/components/research/ResearchContent.tsx apps/web/components/research/research-content.test.tsx
git commit -m "feat(research): ChatThread + ChatComposer + ResearchContent"
```

---

### Task 9: Wire the route + full verification

**Files:**
- Modify: `apps/web/app/(app)/research/page.tsx` (placeholder → real screen)

**Interfaces:**
- Consumes: `ResearchContent` (Task 8), `getApi().getResearchSuggestions` (Task 4), query prefetch helpers (existing).

- [ ] **Step 1: Replace the placeholder page**

Overwrite `apps/web/app/(app)/research/page.tsx`:
```tsx
import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { getQueryClient, qk } from "@/lib/get-query-client";
import { getApi } from "@/lib/api";
import { ResearchContent } from "@/components/research/ResearchContent";

export default async function ResearchPage() {
  const queryClient = getQueryClient();
  await queryClient.prefetchQuery({
    queryKey: qk.researchSuggestions,
    queryFn: () => getApi().getResearchSuggestions(),
  });
  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <ResearchContent />
    </HydrationBoundary>
  );
}
```

- [ ] **Step 2: Run the full test suite**

Run: `npx vitest run`
Expected: all tests pass (previous suite + new research tests).

- [ ] **Step 3: Typecheck**

Run: `npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 4: Production build**

Run: `npm run build`
Expected: success; `/research` appears in the route list (server-rendered, prefetch), no build errors.

- [ ] **Step 5: Commit**

```bash
git add apps/web/app/(app)/research/page.tsx
git commit -m "feat(research): wire /research route (placeholder → ResearchContent)"
```

- [ ] **Step 6: Visual verification (manual, before merge)**

Run `npm run dev`, open `/research`. Confirm: disclaimer banner; suggestion chips on empty thread; clicking a chip streams an answer word-by-word with a blinking cursor; sources render as chips (token/creator/wallet clickable → navigate); composer Enter sends, Shift+Enter newlines, disabled + "Durdur" while streaming; "Durdur" stops mid-stream and keeps partial text. Record findings in `docs/progress.md`.

---

## Self-Review Notes

- **Spec coverage:** §3.1 seam → Task 1+4; §3.2 store → Task 5; §3.3 hook → Task 6; §3.4 registries (`SOURCE_KIND_DEFS`, `pickAnswer`) → Task 2+3; §3.5 mock streaming → Task 4; §3.6 component tree → Task 7+8; §3.7 page/RSC → Task 9; §4 tests → every task; disclaimer label (spec §1) → Task 7. Streaming (simüle), session Zustand persistence, clickable sources all covered.
- **Type consistency:** `addUserMessage → assistantId` used identically in store (Task 5) and hook (Task 6); `streamResearchAnswer(question, onChunk, onDone) => cancel` identical in contract (Task 1), mock (Task 4), hook (Task 6); `ResearchSource.ref`/`kind` consistent across Task 1/2/3/7; `qk.researchSuggestions` added Task 8, consumed Task 8+9.
- **No placeholders:** every code step is complete and runnable. Two accepted interim-state notes (Task 1 tsc, `cn` import path) are explicit, not deferrals.
- **Deferred (spec-marked, not in plan):** real LLM backend, cross-session persist, entity-context binding, multi-conversation history, markdown rendering, copy/export/feedback — intentionally out of scope.
