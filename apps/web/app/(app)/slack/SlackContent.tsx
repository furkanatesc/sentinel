"use client";
import { useNotificationConfig } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";
import SlackConnectionCard from "@/components/slack/SlackConnectionCard";
import NotificationSettings from "@/components/slack/NotificationSettings";
import AlertTemplateList from "@/components/slack/AlertTemplateList";

// SlackContent, /slack ekranının istemci kabuğu: bağlantı + ayarlar + şablonlar
// (sağ sütun mesaj önizlemesi Task 9'da eklenir).
export default function SlackContent() {
  const { data, isLoading, isError } = useNotificationConfig();

  if (isLoading) {
    return (
      <div className="p-6">
        <Skeleton className="h-40 w-full max-w-lg" />
      </div>
    );
  }
  if (isError || !data) {
    return <div className="p-6 text-sm text-critical">Slack yapılandırması alınamadı.</div>;
  }

  return (
    <div className="p-6">
      <h1 className="mb-1 text-lg font-semibold text-foreground">Slack</h1>
      <p className="mb-4 text-sm text-foreground/60">Bildirim teslimat kanalı ve şablonları.</p>
      <div className="max-w-lg space-y-4">
        <SlackConnectionCard config={data} />
        <NotificationSettings config={data} />
        <AlertTemplateList templates={data.templates} />
      </div>
    </div>
  );
}
