"use client";
import { useEffect, useRef } from "react";
import cytoscape, { type Core } from "cytoscape";
import type { CyElement } from "@/lib/graph/elements";
import type { CyStyle } from "@/lib/graph/stylesheet";

export function WalletGraphCanvas({ elements, stylesheet, onNodeSelect }: {
  elements: CyElement[]; stylesheet: CyStyle[]; onNodeSelect: (id: string | null) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const cbRef = useRef(onNodeSelect);
  cbRef.current = onNodeSelect;
  useEffect(() => {
    if (!ref.current) return;
    const cy: Core = cytoscape({
      container: ref.current,
      elements: elements as cytoscape.ElementDefinition[],
      style: stylesheet as cytoscape.StylesheetJsonBlock[],
      layout: { name: "cose", animate: false, padding: 30 },
      wheelSensitivity: 0.2,
    });
    cy.on("tap", "node", (evt) => cbRef.current(evt.target.id()));
    cy.on("tap", (evt) => { if (evt.target === cy) cbRef.current(null); });
    return () => cy.destroy();
  }, [elements, stylesheet]);
  return <div ref={ref} data-testid="graph-canvas" className="h-[600px] w-full rounded-lg border border-border bg-card" />;
}
