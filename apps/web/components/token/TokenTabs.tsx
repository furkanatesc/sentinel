"use client";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { TAB_DEFS } from "./tab-defs";
import { OverviewTab } from "./tabs/OverviewTab";
import { RiskAnalysisTab } from "./tabs/RiskAnalysisTab";
import type { TokenDetail } from "@/lib/api/types";

export function TokenTabs({ token }: { token: TokenDetail }) {
  return (
    <Tabs defaultValue="overview" className="w-full">
      <TabsList className="flex h-auto flex-wrap">
        {TAB_DEFS.map((t) => <TabsTrigger key={t.key} value={t.key}>{t.label}</TabsTrigger>)}
      </TabsList>
      <TabsContent value="overview" className="mt-4"><OverviewTab token={token} /></TabsContent>
      <TabsContent value="risk" className="mt-4"><RiskAnalysisTab risks={token.risks} /></TabsContent>
      {TAB_DEFS.filter((t) => !t.built).map((t) => (
        <TabsContent key={t.key} value={t.key} className="mt-4">
          <div className="rounded-lg border border-dashed border-border bg-card py-16 text-center text-muted-foreground" style={{ fontSize: 13 }}>{t.label} — yakında</div>
        </TabsContent>
      ))}
    </Tabs>
  );
}
