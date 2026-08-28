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
