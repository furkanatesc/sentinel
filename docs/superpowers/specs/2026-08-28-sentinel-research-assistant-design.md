# SENTINEL Frontend — Research Assistant Design Spec
### Ekran 11: Research Assistant (AI destekli araştırma paneli)

- **Tarih:** 2026-08-28
- **Durum:** Tasarım onaylandı (kullanıcı, 2026-08-28) — branch `feat/research-assistant`. Implementasyon plana bekliyor.
- **Önkoşul:** Increment 1–9 master'da; mock seam (`lib/api/`), Zustand store deseni (`lib/store/`), shadcn/Base-UI primitives.
- **Kaynaklar:** `docs/design/sentinel-ui-ux-design.md` (Ekran 11: Research Assistant), `MEMORY.md` (Alerts/Research veri-blokusuz artım), `docs/progress.md`.

---

## 1. Amaç ve kapsam

AI destekli, **çok-turlu sohbet** araştırma paneli. Kullanıcı doğal-dil soru sorar → asistan
**kaynaklı** (tx hash, wallet, token, risk rule, strategy version, timestamp) bir cevap **kelime-kelime
akıtarak** (simüle streaming) döndürür. Cevaplar **"informational analysis"** olarak etiketlenir ve
trade kararından ayrıştırılır (tasarım spec zorunluluğu).

**Rota:** Mevcut **"Araştırma" / `/research`** nav placeholder'ı (nav.ts satır 24, `Bot` ikonu) gerçeğe döner (rename yok).

**Yerleşim:** tek sütun sohbet paneli. Üstte `InfoDisclaimerBanner`; ortada `ChatThread` (bounded scroll +
auto-scroll, Live Feed `FeedTable` deseni); thread boşken `SuggestionChips` (başlangıç örnek soruları);
altta yapışık `ChatComposer` (textarea + Gönder/Stop). Dar viewport'ta tek sütun korunur.

**Backend yok — tam simüle (deterministik canned Q→A):** "Çalıştır" backend'i yok. `streamResearchAnswer`
seam'i mevcut `subscribe*(cb) => unsubscribe` desenini birebir taklit eder: soruyu saf `pickAnswer` ile
anahtar-kelimeye göre canned cevaba yönlendirir, cevabı `setInterval` (~35ms) ile kelime-kelime `onChunk`'a
yollar, bitince `onDone`'a yapılandırılmış kaynakları verir. Eşleşme yoksa genel fallback cevabı.

**Etkileşim kararları (kullanıcı onaylı, 2026-08-28):**
- Çok-turlu **sohbet** (tek-atış Q→A değil).
- **Simüle streaming** (parça parça, typing efekti).
- **Oturum-boyu** persist (Zustand; reload'da sıfırlanır, localStorage yok).
- **Tıklanabilir** kaynaklar (uygulama-içi navigasyon).

**Kapsam dışı (bilinçli — işaretli, sessiz düşürme yok):**
- Gerçek LLM backend (mock; seam hazır, http → `notReady`).
- Cross-session persist (localStorage/DB yok — oturum-boyu Zustand yeterli).
- Varlık-context binding (belirli token/creator seçip soruları ona çözme — düz sohbet seçildi).
- Çoklu / kayıtlı konuşma geçmişi, konuşma listesi sidebar'ı.
- Markdown/zengin-metin render (düz metin paragrafları + kaynak çipleri yeterli).
- Kopyala / export / regenerate / thumbs feedback butonları.
- RHF+Zod (composer basit kontrollü textarea + saf gönderilebilirlik kontrolü yeterli).

---

## 2. Clean Code & SOLID + reuse (ölçüt)

- **Reuse:**
  - `next/link` `Link` — kaynak navigasyonu (`/tokens/[mint]`, `/creators/[address]`, `/wallet-graph`).
  - Zustand `create` deseni (`lib/store/session.ts`, `ui.ts`) → yeni `research` store.
  - Bounded-scroll + auto-scroll → Live Feed `FeedTable` yaklaşımı.
  - shadcn/Base-UI `Button`, `ScrollArea`/native scroll, ikon seti (lucide) — mevcut.
  - `cn` util + Sentinel dark tema token'ları.
- **SRP:** her bileşen tek iş (`ChatThread` liste, `ChatMessageBubble` tek mesaj, `SourceChip` tek kaynak,
  `ChatComposer` girdi, `SuggestionChips` başlangıç, `InfoDisclaimerBanner` etiket).
- **OCP:** `SOURCE_KIND_DEFS` registry (yeni kaynak türü = registry'ye satır; bileşen değişmez);
  suggestion listesi + canned-answer havuzu data-driven.
- **DIP:** bileşenler `useResearch()` hook'unu, hook `getApi()`'yi görür; **store mock/http import ETMEZ**
  (saf state+reducer). Hiçbir bileşen mock'u doğrudan import etmez.
- **ISP:** dar prop'lar — `ChatMessageBubble` yalnız tek `ChatMessage`, `SourceChip` yalnız tek `ResearchSource`.

---

## 3. Mimari

### 3.1 Veri seam genişlemesi (`lib/api`)

**`types.ts` (yeni tipler):**
```ts
export type ResearchSourceKind =
  | "token" | "creator" | "wallet" | "tx" | "risk-rule" | "strategy" | "timestamp";

export interface ResearchSource {
  id: string;
  kind: ResearchSourceKind;
  label: string;        // gösterilecek etiket (ör. "AERO", "6Rt4…9kQ", "rug-pull-rule")
  ref?: string;         // hedef id (mint | address | tx sig | strategy id); href presentation'da kurulur
}

export interface ResearchAnswer {
  text: string;                 // tam cevap metni
  sources: ResearchSource[];
}

export interface ResearchSuggestion {
  id: string;
  text: string;                 // örnek başlangıç sorusu
}
```

**`contract.ts` — `SentinelApi`'ye 2 metot:**
```ts
getResearchSuggestions(): Promise<ResearchSuggestion[]>;
/** Streaming seam — subscribe deseni. onChunk incremental metin, onDone final+kaynaklar. Cancel fn döner. */
streamResearchAnswer(
  question: string,
  onChunk: (chunk: string) => void,
  onDone: (answer: ResearchAnswer) => void,
): () => void;
```

**`http.ts`:** ikisi de `notReady("getResearchSuggestions")` / `notReady("streamResearchAnswer")`
(diğer canlı-olmayan endpoint deseni). `streamResearchAnswer` senkron `notReady` fırlatır (Promise değil);
canlı gelince SSE/WS ile doldurulur. `LIVE_ENDPOINTS`'e EKLENMEZ (mock kalır).

### 3.2 Client sohbet modeli + store (`lib/store/research.ts`)

Store **saf state + reducer** (I/O yok, mock/http import yok → DIP):
```ts
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
  // reducers (saf, test edilebilir):
  addUserMessage(text: string): string;        // döner: yeni assistant placeholder id
  appendChunk(assistantId: string, chunk: string): void;
  finishAnswer(assistantId: string, sources: ResearchSource[]): void;
  reset(): void;
}
```
`addUserMessage` hem user mesajını hem boş `status:"streaming"` assistant placeholder'ını ekler,
`isStreaming=true` yapar; assistant id döner. `appendChunk` metni büyütür. `finishAnswer` kaynakları
ekler + `status:"done"` + `isStreaming=false`.

### 3.3 Orkestrasyon hook (`components/research/use-research.ts` veya `lib/research/`)

`useResearch()` — seam ile store arası köprü (DIP: getApi burada çağrılır):
```ts
function useResearch() {
  const store = useResearchStore();
  const cancelRef = useRef<null | (() => void)>(null);
  const send = (question: string) => {
    if (store.isStreaming) return;
    const assistantId = store.addUserMessage(question);
    cancelRef.current = getApi().streamResearchAnswer(
      question,
      (chunk) => store.appendChunk(assistantId, chunk),
      (answer) => { store.finishAnswer(assistantId, answer.sources); cancelRef.current = null; },
    );
  };
  const stop = () => { cancelRef.current?.(); cancelRef.current = null; /* store.finishAnswer boş kaynak */ };
  // unmount'ta cancel (cleanup)
  return { messages: store.messages, isStreaming: store.isStreaming, send, stop };
}
```
`onDone` cancel yerine normal biterse `finishAnswer` metni akmış haliyle kalır. `stop` erken durdurma:
akmış metni koruyup `status:"done"` yapar (kaynak `[]`).

### 3.4 OCP registry + saf mantık (`lib/research/`)

- **`source-defs.ts`:** `SOURCE_KIND_DEFS: Record<ResearchSourceKind, { label; icon; buildHref?(src) }>`.
  - `token` → `/tokens/${ref}`, `creator` → `/creators/${ref}`, `wallet` → `/wallet-graph` (opsiyonel `?focus=${ref}`).
  - `tx` / `risk-rule` / `strategy` / `timestamp` → `buildHref` yok → linksiz etiket çipi.
    (Not: `strategy` için `/strategies/${ref}` linki de eklenebilir — plan aşamasında değerlendirilecek; şimdilik linksiz.)
- **`match.ts`:** saf `pickAnswer(question: string): CannedAnswer` — normalize edilmiş soruyu anahtar-kelime
  gruplarıyla eşler (ör. "risk|neden riskli" → risk cevabı; "creator|üretici|geçmiş" → creator cevabı;
  "cluster|küme|wallet|cüzdan" → wallet cluster; "benzer|performans" → benzer-token; "sinyal|neden üretildi"
  → sinyal). Eşleşme yoksa `FALLBACK_ANSWER`. `CannedAnswer = { text; sources: ResearchSource[] }`.

### 3.5 Mock (`mock.ts`)

```ts
getResearchSuggestions() { return delay(RESEARCH_SUGGESTIONS); }   // spec örnek soruları

streamResearchAnswer(question, onChunk, onDone) {
  const answer = pickAnswer(question);          // saf, lib/research/match
  const words = answer.text.split(" ");
  let i = 0;
  const id = setInterval(() => {
    if (i < words.length) { onChunk((i === 0 ? "" : " ") + words[i]); i++; }
    else { clearInterval(id); onDone({ text: answer.text, sources: answer.sources }); }
  }, 35);
  return () => clearInterval(id);
}
```
`RESEARCH_SUGGESTIONS` + canned havuz (`match.ts` içinde veya mock-yerel data). Kaynaklar mevcut mock
token symbol'leri / wallet adresleriyle tutarlı (tıklanınca gerçek mock ekrana gider).

### 3.6 Bileşen ağacı (`components/research/`)

```
ResearchContent (page client — "use client")
├─ InfoDisclaimerBanner            ("Bu analizler bilgilendirme amaçlıdır; trade kararı değildir.")
├─ ChatThread                      (bounded scroll + auto-scroll bottom)
│  ├─ SuggestionChips              (yalnız messages boşken → send(chip.text))
│  └─ ChatMessageBubble[]          (role'e göre user/assistant stil; assistant streaming'de imleç)
│     └─ SourceList → SourceChip[] (yalnız assistant, status:"done", sources.length)
└─ ChatComposer                    (kontrollü textarea; Enter=gönder, Shift+Enter=satır;
                                    isStreaming'de input disabled + Gönder→Stop)
```
- `ChatThread` yeni mesajda/aktif chunk'ta `scrollIntoView` (Live Feed auto-scroll deseni).
- `ChatMessageBubble` streaming imleç: `status:"streaming"` iken metin sonuna blink `▍`.
- `SourceChip`: `SOURCE_KIND_DEFS[kind].buildHref` varsa `<Link>`, yoksa `<span>` çip; ikon + label.

### 3.7 RSC / sayfa

`app/(app)/research/page.tsx`: hafif — suggestions RSC prefetch opsiyonel (statik liste; küçük).
`ResearchContent` client component (`"use client"`), store + hook + streaming client-side. Sohbet
tamamen client-state (Zustand); React Query'ye gerek yok (streaming callback-driven). Suggestions için
istersek basit `useQuery(qk.researchSuggestions, getResearchSuggestions)` — plan aşamasında karar.

---

## 4. Test stratejisi (Vitest + RTL)

**Saf mantık:**
- `pickAnswer`: her anahtar-kelime grubu doğru cevaba, bilinmeyen → fallback (non-vacuous).
- `SOURCE_KIND_DEFS` `buildHref`: token/creator/wallet doğru href; tx/risk/timestamp → undefined.
- Store reducer'ları: `addUserMessage` iki mesaj + isStreaming; `appendChunk` birikimli; `finishAnswer`
  kaynak+done+isStreaming false; `reset`.

**Seam / mock:**
- `streamResearchAnswer` (fake timers): chunk'lar sırayla gelir, sonra `onDone` tam metin+kaynak;
  cancel fn `onDone`'u engeller (clearInterval).
- `getResearchSuggestions` liste döner.

**Bileşen (RTL):**
- `SuggestionChips` yalnız boş thread'de görünür; tıklama `send` tetikler.
- `ChatComposer` Enter gönderir, isStreaming'de disabled + Stop görünür.
- `SourceChip` link kaynağı doğru `href`; linksiz kind `<a>` render etmez.
- `ChatThread` mesaj listesini + streaming imleci render eder.

**Hedef:** mevcut yeşil suite'e ek testler; `npm run build` `/research` sayfası başarılı.

---

## 5. Dosya değişiklikleri (özet)

**Yeni:**
- `lib/store/research.ts` (Zustand, saf)
- `lib/research/source-defs.ts`, `lib/research/match.ts` (+ testleri)
- `components/research/ResearchContent.tsx`, `ChatThread.tsx`, `ChatMessageBubble.tsx`,
  `SourceList.tsx`/`SourceChip.tsx`, `SuggestionChips.tsx`, `ChatComposer.tsx`,
  `InfoDisclaimerBanner.tsx`, `use-research.ts` (+ testleri)

**Değişen:**
- `lib/api/types.ts` (+ Research tipleri)
- `lib/api/contract.ts` (+ 2 metot)
- `lib/api/mock.ts` (+ getResearchSuggestions, streamResearchAnswer, RESEARCH_SUGGESTIONS)
- `lib/api/http.ts` (+ 2 notReady)
- `app/(app)/research/page.tsx` (placeholder → ResearchContent)

**Değişmeyen:** `nav.ts` (rota zaten var), `index.ts` seam (yeni LIVE_ENDPOINTS yok).

---

## 6. Riskler / açık noktalar (plan aşamasında)

- Streaming imleç + auto-scroll performansı uzun cevaplarda (35ms tick; kısa canned metinlerde sorun yok).
- `strategy` kaynak linki: şimdilik linksiz; `/strategies/[id]` reuse edilebilir (plan kararı).
- Suggestions RSC prefetch vs client `useQuery` (statik liste; küçük — plan kararı, YAGNI eğilimi client-yok).
- `stop` (erken durdurma) UX: akmış metni koru + done. Store `finishAnswer(assistantId, [])` reuse.
