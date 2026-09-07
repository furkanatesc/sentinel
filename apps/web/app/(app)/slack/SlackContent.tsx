"use client";
import { useNotificationConfig } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";
import SlackConnectionCard from "@/components/slack/SlackConnectionCard";
import NotificationSettings from "@/components/slack/NotificationSettings";
import AlertTemplateList from "@/components/slack/AlertTemplateList";
import SlackMessagePreview from "@/components/slack/SlackMessagePreview";

// SlackContent, /slack ekranının istemci kabuğu: sol sütun bağlantı + ayarlar + şablonlar,
// sağ sütun canlı Slack mesaj önizlemesi.
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
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,360px)]">
        <div className="space-y-4">
          <SlackConnectionCard config={data} />
          <NotificationSettings config={data} />
          <AlertTemplateList templates={data.templates} />
        </div>
        <SlackMessagePreview config={data} />
      </div>
    </div>
  );
}
