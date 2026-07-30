"use client";
import { useEffect, useRef } from "react";
import cytoscape, { type Core } from "cytoscape";
import type { CyElement } from "@/lib/graph/elements";
import type { CyStyle } from "@/lib/graph/stylesheet";

export function WalletGraphCanvas({ elements, stylesheet, onNodeSelect, focusNodeId }: {
  elements: CyElement[]; stylesheet: CyStyle[]; onNodeSelect: (id: string | null) => void; focusNodeId: string | null;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const cbRef = useRef(onNodeSelect);
  const cyRef = useRef<Core | null>(null);
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
    cyRef.current = cy;
    return () => {
      cyRef.current = null;
      cy.destroy();
    };
  }, [elements, stylesheet]);

  useEffect(() => {
    const cy = cyRef.current;
    if (!cy) return;
    cy.batch(() => {
      cy.elements().removeClass("faded");
      if (focusNodeId) {
        const n = cy.getElementById(focusNodeId);
        const keep = n.closedNeighborhood();
        cy.elements().not(keep).addClass("faded");
      }
    });
  }, [focusNodeId]);

  return <div ref={ref} data-testid="graph-canvas" className="h-[600px] w-full rounded-lg border border-border bg-card" />;
}
