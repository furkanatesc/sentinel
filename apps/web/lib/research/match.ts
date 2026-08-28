import type { ResearchSource, ResearchSuggestion, ResearchAnswer } from "@/lib/api/types";

export type CannedAnswer = ResearchAnswer;

interface Rule { keywords: string[]; answer: CannedAnswer; }

// source refs resolve to real mock.ts entities: token symbols (GFROG/HLS/PULSE/ZAP),
// creator addrs (CREATOR_ADDRS), wallet-graph node addrs, strategy ids (STRATEGY_DEFS).
const RULES: Rule[] = [
  {
    keywords: ["riskli", "risk", "neden riskli", "tehlike"],
    answer: {
      text: "GFROG yüksek riskli görünüyor: üretici likidite havuzunun %92'sini çekti ve ilk 5 cüzdan arzın %78'ini elinde tutuyor. Bu, rug-pull kalıbıyla eşleşiyor; güvenlik skoru 22/100.",
      sources: [
        { id: "r1", kind: "token", label: "GFROG", ref: "GFROG" },
        { id: "r2", kind: "risk-rule", label: "likidite-cekildi" },
        { id: "r3", kind: "wallet", label: "Şüpheli-1", ref: "Sus1dd" },
      ],
    },
  },
  {
    keywords: ["üretici", "uretici", "creator", "geçmiş", "gecmis"],
    answer: {
      text: "Bu üreticinin 6 önceki tokenından 4'ü rug ile sonuçlandı, 1'i graduate oldu. Ortalama likidite ömrü 3 saat. Üretici itibar skoru 29/100.",
      sources: [
        { id: "c1", kind: "creator", label: "Creator-C", ref: "CreCqw" },
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
        { id: "w1", kind: "wallet", label: "Funder-1", ref: "Fnd1Qk" },
        { id: "w2", kind: "tx", label: "5xAr…t9Kp" },
        { id: "w3", kind: "token", label: "GFROG", ref: "GFROG" },
      ],
    },
  },
  {
    keywords: ["benzer", "performans"],
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
      text: "Sinyal, momentum 88'e çıkıp güvenlik skoru 78'in üstünde kalınca 'Momentum Scalp' stratejisi tarafından üretildi. Tetikleyici: 5dk hacim > $40K.",
      sources: [
        { id: "s1", kind: "strategy", label: "Momentum Scalp", ref: "momentum-scalp" },
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
