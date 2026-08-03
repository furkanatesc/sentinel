import {
  LayoutDashboard, Radio, Compass, Coins, UserSearch, Share2, Sparkles,
  Layers, Briefcase, Terminal, PieChart, History, Bell, Send, Bot,
  Activity, Settings, type LucideIcon,
} from "lucide-react";

export interface NavItem { label: string; path: string; icon: LucideIcon; }

export const navItems: NavItem[] = [
  { label: "Genel Bakış", path: "/", icon: LayoutDashboard },
  { label: "Canlı Akış", path: "/live-feed", icon: Radio },
  { label: "Keşfet", path: "/discover", icon: Compass },
  { label: "Tokenlar", path: "/tokens", icon: Coins },
  { label: "Üreticiler", path: "/creators", icon: UserSearch },
  { label: "Cüzdan Grafiği", path: "/wallet-graph", icon: Share2 },
  { label: "Akıllı Cüzdanlar", path: "/smart-wallets", icon: Sparkles },
  { label: "Stratejiler", path: "/strategies", icon: Layers },
  { label: "Pozisyonlar", path: "/positions", icon: Briefcase },
  { label: "Terminal", path: "/terminal", icon: Terminal },
  { label: "Portföy", path: "/portfolio", icon: PieChart },
  { label: "Geriye Test", path: "/backtesting", icon: History },
  { label: "Uyarılar", path: "/alerts", icon: Bell },
  { label: "Telegram", path: "/telegram", icon: Send },
  { label: "Araştırma", path: "/research", icon: Bot },
  { label: "Sistem Sağlığı", path: "/system-health", icon: Activity },
  { label: "Ayarlar", path: "/settings", icon: Settings },
];
