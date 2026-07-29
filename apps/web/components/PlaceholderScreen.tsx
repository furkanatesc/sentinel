"use client";
import { usePathname } from "next/navigation";
import { Construction } from "lucide-react";
import { navItems } from "@/components/shell/nav";

export function PlaceholderScreen() {
  const pathname = usePathname();
  const item = navItems.find((n) => n.path === pathname);
  const Icon = item?.icon ?? Construction;
  return (
    <div className="space-y-5">
      <div>
        <h1>{item?.label ?? "Screen"}</h1>
        <p className="text-muted-foreground" style={{ fontSize: 13 }}>This module is part of the Sentinel platform blueprint.</p>
      </div>
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border bg-card py-24 text-center">
        <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-accent">
          <Icon size={26} className="text-primary" />
        </div>
        <h3 className="mb-1">{item?.label} coming soon</h3>
        <p className="mb-5 max-w-sm text-muted-foreground" style={{ fontSize: 13 }}>
          The Overview Dashboard is fully built out. Tell me which screen to detail next.
        </p>
      </div>
    </div>
  );
}
