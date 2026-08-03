"use client";
import { useTransactions } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";

const KIND_LABEL: Record<string, string> = { buy: "Alım", sell: "Satım", approve: "Onay" };
const STATUS_COLOR: Record<string, string> = { success: "#2FD98B", pending: "#FFB020", failed: "#F0476B" };
const STATUS_LABEL: Record<string, string> = { success: "Başarılı", pending: "Bekliyor", failed: "Başarısız" };

export function TransactionsTable() {
  const { data, isLoading } = useTransactions();
  if (isLoading || !data) return <Skeleton className="h-40 w-full" />;
  if (data.length === 0) return <div className="p-6 text-center text-muted-foreground" style={{ fontSize: 13 }}>İşlem yok</div>;
  return (
    <table className="w-full" style={{ fontSize: 12 }}>
      <thead className="text-muted-foreground">
        <tr>
          <th className="px-3 py-2 text-left">Hash</th><th className="px-3 py-2 text-left">Tür</th>
          <th className="px-3 py-2 text-left">Token</th><th className="px-3 py-2 text-right">Tutar</th>
          <th className="px-3 py-2 text-left">Durum</th><th className="px-3 py-2 text-right">Zaman</th>
        </tr>
      </thead>
      <tbody>
        {data.map((t) => (
          <tr key={t.id} className="border-t border-border">
            <td className="px-3 py-2">
              <a href={`https://solscan.io/tx/${t.hash}`} target="_blank" rel="noreferrer" className="text-primary hover:underline">{t.hash}</a>
            </td>
            <td className="px-3 py-2">{KIND_LABEL[t.kind]}</td>
            <td className="px-3 py-2 font-medium">{t.tokenSymbol}</td>
            <td className="px-3 py-2 text-right">{t.amountSol} SOL</td>
            <td className="px-3 py-2" style={{ color: STATUS_COLOR[t.status] }}>{STATUS_LABEL[t.status]}</td>
            <td className="px-3 py-2 text-right text-muted-foreground">{t.time}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
