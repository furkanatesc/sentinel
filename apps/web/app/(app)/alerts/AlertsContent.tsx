"use client";
import { useState } from "react";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import AlertRulesPanel from "@/components/alerts/AlertRulesPanel";
import AlertHistoryPanel from "@/components/alerts/AlertHistoryPanel";
import AlertRuleForm from "@/components/alerts/AlertRuleForm";

// AlertsContent, /alerts ekranının istemci kabuğu: "Kurallar" | "Geçmiş" sekmeleri +
// "Yeni Kural" formu (Sheet). Kural oluşturma simüle (bkz AlertRuleForm).
export default function AlertsContent() {
  const [formOpen, setFormOpen] = useState(false);
  return (
    <div className="p-6">
      <h1 className="mb-1 text-lg font-semibold text-foreground">Uyarılar</h1>
      <p className="mb-4 text-sm text-foreground/60">Alarm kuralları ve geçmiş bildirimler.</p>
      <Tabs defaultValue="rules">
        <TabsList>
          <TabsTrigger value="rules">Kurallar</TabsTrigger>
          <TabsTrigger value="history">Geçmiş</TabsTrigger>
        </TabsList>
        <TabsContent value="rules" className="pt-4">
          <AlertRulesPanel onNew={() => setFormOpen(true)} />
        </TabsContent>
        <TabsContent value="history" className="pt-4">
          <AlertHistoryPanel />
        </TabsContent>
      </Tabs>
      <Sheet open={formOpen} onOpenChange={setFormOpen}>
        <SheetContent side="right" className="w-[380px] overflow-y-auto bg-popover">
          <SheetHeader>
            <SheetTitle>Yeni Alarm Kuralı</SheetTitle>
          </SheetHeader>
          <div className="mt-4">
            <AlertRuleForm onDone={() => setFormOpen(false)} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}
