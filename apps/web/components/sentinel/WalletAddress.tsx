"use client";
import { useState } from "react";
import { Copy, Check, ExternalLink } from "lucide-react";
import { toast } from "sonner";

interface WalletAddressProps { address: string; explorer?: boolean; }

export function WalletAddress({ address, explorer = true }: WalletAddressProps) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard?.writeText(address);
    setCopied(true);
    toast.success("Address copied", { description: address });
    setTimeout(() => setCopied(false), 1400);
  };
  return (
    <span className="group inline-flex items-center gap-1.5">
      <span className="font-mono text-muted-foreground" style={{ fontSize: 12 }}>{address}</span>
      <button onClick={copy} className="text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100" title="Copy address">
        {copied ? <Check size={13} className="text-positive" /> : <Copy size={13} />}
      </button>
      {explorer && (
        <a href="#" onClick={(e) => e.preventDefault()} className="text-muted-foreground opacity-0 transition-opacity hover:text-foreground group-hover:opacity-100" title="View on explorer">
          <ExternalLink size={13} />
        </a>
      )}
    </span>
  );
}
