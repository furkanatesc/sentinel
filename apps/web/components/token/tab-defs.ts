export interface TabDef { key: string; label: string; built: boolean; }

export const TAB_DEFS: TabDef[] = [
  { key: "overview", label: "Genel Bakış", built: true },
  { key: "risk", label: "Risk Analizi", built: true },
  { key: "market", label: "Piyasa", built: false },
  { key: "holders", label: "Sahipler", built: false },
  { key: "creator", label: "Üretici", built: false },
  { key: "wallet-graph", label: "Cüzdan Grafiği", built: false },
  { key: "transactions", label: "İşlemler", built: false },
  { key: "social", label: "Sosyal", built: false },
  { key: "signals", label: "Strateji Sinyalleri", built: false },
  { key: "audit", label: "Denetim Günlüğü", built: false },
];
