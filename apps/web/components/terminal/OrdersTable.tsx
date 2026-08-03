"use client";
import { toast } from "sonner";
import { useOrders } from "@/lib/hooks/queries";
import { ORDER_SIDE_DEFS, ORDER_TYPE_DEFS } from "@/lib/terminal/order-defs";
import { Skeleton } from "@/components/ui/skeleton";

const STATUS_LABEL: Record<string, string> = { open: "Açık", filled: "Doldu", cancelled: "İptal" };

export function OrdersTable() {
  const { data, isLoading } = useOrders();
  if (isLoading || !data) return <Skeleton className="h-40 w-full" />;
  if (data.length === 0) return <div className="p-6 text-center text-muted-foreground" style={{ fontSize: 13 }}>Emir yok</div>;
  const cancel = (sym: string) => toast(`Emir iptal — ${sym}`, { description: "Bu demoda simüle edilir." });

  return (
    <table className="w-full" style={{ fontSize: 12 }}>
      <thead className="text-muted-foreground">
        <tr>
          <th className="px-3 py-2 text-left">Token</th><th className="px-3 py-2 text-left">Yön</th>
          <th className="px-3 py-2 text-left">Tür</th><th className="px-3 py-2 text-left">Durum</th>
          <th className="px-3 py-2 text-right">Fiyat</th><th className="px-3 py-2 text-right">Miktar</th>
          <th className="px-3 py-2 text-right">Aksiyon</th>
        </tr>
      </thead>
      <tbody>
        {data.map((o) => {
          const side = ORDER_SIDE_DEFS.find((s) => s.key === o.side)!;
          const type = ORDER_TYPE_DEFS.find((t) => t.key === o.type)!;
          return (
            <tr key={o.id} className="border-t border-border">
              <td className="px-3 py-2 font-medium">{o.tokenSymbol}</td>
              <td className="px-3 py-2" style={{ color: side.color }}>{side.label}</td>
              <td className="px-3 py-2">{type.label}</td>
              <td className="px-3 py-2">{STATUS_LABEL[o.status]}</td>
              <td className="px-3 py-2 text-right">{o.price}</td>
              <td className="px-3 py-2 text-right">{o.amountSol} SOL</td>
              <td className="px-3 py-2 text-right">
                {o.status === "open" && (
                  <button onClick={() => cancel(o.tokenSymbol)} className="rounded-md px-2 py-1" style={{ color: "#F0476B", border: "1px solid rgba(240,71,107,0.4)" }}>İptal</button>
                )}
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}
