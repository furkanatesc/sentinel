import { Info } from "lucide-react";

export function InfoDisclaimerBanner() {
  return (
    <div className="flex items-center gap-2 rounded-md border border-border/60 bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
      <Info className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
      <span>Bu analizler yalnızca bilgilendirme amaçlıdır; trade kararı ya da yatırım tavsiyesi değildir.</span>
    </div>
  );
}
