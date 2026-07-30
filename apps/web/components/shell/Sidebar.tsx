"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronLeft, ShieldCheck, Wifi, Send } from "lucide-react";
import { navItems } from "./nav";
import { TradingModeBadge } from "./TradingModeBadge";
import { useUiStore } from "@/lib/store/ui";

export function Sidebar() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const onToggle = useUiStore((s) => s.toggleSidebar);
  const pathname = usePathname();
  return (
    <aside className="flex h-full flex-col border-r border-sidebar-border bg-sidebar transition-all duration-200" style={{ width: collapsed ? 68 : 232 }}>
      <div className="flex h-14 items-center gap-2.5 border-b border-sidebar-border px-4">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary">
          <ShieldCheck size={18} className="text-primary-foreground" />
        </div>
        {!collapsed && (
          <div className="flex flex-col leading-tight">
            <span style={{ fontSize: 15, fontWeight: 600 }}>Sentinel</span>
            <span className="text-muted-foreground" style={{ fontSize: 10 }}>Solana Intelligence</span>
          </div>
        )}
        <button onClick={onToggle} className="ml-auto text-muted-foreground transition-colors hover:text-foreground" title={collapsed ? "Expand" : "Collapse"}>
          <ChevronLeft size={16} className={collapsed ? "rotate-180" : ""} />
        </button>
      </div>
      <nav className="flex-1 overflow-y-auto px-2 py-3">
        {navItems.map((item) => {
          const isActive = item.path === "/" ? pathname === "/" : pathname.startsWith(item.path);
          const Icon = item.icon;
          return (
            <Link key={item.path} href={item.path} title={collapsed ? item.label : undefined}
              className={`flex items-center gap-3 rounded-md px-2.5 py-2 transition-colors ${isActive ? "bg-sidebar-accent text-sidebar-accent-foreground" : "text-sidebar-foreground hover:bg-sidebar-accent/50 hover:text-foreground"}`}>
              <Icon size={17} className={isActive ? "text-primary" : ""} />
              {!collapsed && <span style={{ fontSize: 13 }}>{item.label}</span>}
            </Link>
          );
        })}
      </nav>
      <div className="border-t border-sidebar-border p-3">
        {!collapsed && (
          <div className="mb-3 space-y-1.5">
            <StatusRow icon={<Wifi size={12} />} label="RPC" value="142 ms" ok />
            <StatusRow icon={<span className="h-2 w-2 rounded-full bg-positive" />} label="Solana" value="Healthy" ok />
            <StatusRow icon={<Send size={12} />} label="Telegram" value="Connected" ok />
          </div>
        )}
        <TradingModeBadge collapsed={collapsed} />
      </div>
    </aside>
  );
}

function StatusRow({ icon, label, value, ok }: { icon: React.ReactNode; label: string; value: string; ok?: boolean }) {
  return (
    <div className="flex items-center gap-2 text-muted-foreground" style={{ fontSize: 11 }}>
      <span className={ok ? "text-positive" : "text-critical"}>{icon}</span>
      <span>{label}</span>
      <span className="ml-auto font-mono text-foreground">{value}</span>
    </div>
  );
}
