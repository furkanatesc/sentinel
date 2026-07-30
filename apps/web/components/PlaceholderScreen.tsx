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
        <h1>{item?.label ?? "Ekran"}</h1>
        <p className="text-muted-foreground" style={{ fontSize: 13 }}>Bu modül Sentinel platform planının bir parçası.</p>
      </div>
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border bg-card py-24 text-center">
        <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-accent">
          <Icon size={26} className="text-primary" />
        </div>
        <h3 className="mb-1">{item?.label} yakında</h3>
        <p className="mb-5 max-w-sm text-muted-foreground" style={{ fontSize: 13 }}>
          Genel Bakış paneli tamamen hazır. Sıradaki ekranı detaylandırmamı söyle.
        </p>
      </div>
    </div>
  );
}
