import {
  LayoutDashboard, Radio, Compass, Coins, UserSearch, Share2, Sparkles,
  Layers, Briefcase, ListOrdered, PieChart, History, Bell, Send, Bot,
  Activity, Settings, type LucideIcon,
} from "lucide-react";

export interface NavItem { label: string; path: string; icon: LucideIcon; }

export const navItems: NavItem[] = [
  { label: "Overview", path: "/", icon: LayoutDashboard },
  { label: "Live Feed", path: "/live-feed", icon: Radio },
  { label: "Discover", path: "/discover", icon: Compass },
  { label: "Tokens", path: "/tokens", icon: Coins },
  { label: "Creators", path: "/creators", icon: UserSearch },
  { label: "Wallet Graph", path: "/wallet-graph", icon: Share2 },
  { label: "Smart Wallets", path: "/smart-wallets", icon: Sparkles },
  { label: "Strategies", path: "/strategies", icon: Layers },
  { label: "Positions", path: "/positions", icon: Briefcase },
  { label: "Orders", path: "/orders", icon: ListOrdered },
  { label: "Portfolio", path: "/portfolio", icon: PieChart },
  { label: "Backtesting", path: "/backtesting", icon: History },
  { label: "Alerts", path: "/alerts", icon: Bell },
  { label: "Telegram", path: "/telegram", icon: Send },
  { label: "Research", path: "/research", icon: Bot },
  { label: "System Health", path: "/system-health", icon: Activity },
  { label: "Settings", path: "/settings", icon: Settings },
];
