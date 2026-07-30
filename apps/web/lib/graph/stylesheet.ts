import { NODE_TYPE_DEFS, EDGE_TYPE_DEFS } from "./graph-defs";
import { riskMeta } from "@/lib/format";

export interface CyStyle { selector: string; style: Record<string, unknown>; }

export function buildStylesheet(): CyStyle[] {
  const styles: CyStyle[] = [
    { selector: "node", style: { label: "data(label)", color: "#C4CBD8", "font-size": 9, "text-valign": "bottom", "text-margin-y": 4, width: 26, height: 26, "border-width": 2, "border-color": "rgba(255,255,255,0.15)" } },
    { selector: "edge", style: { width: 1.5, "curve-style": "bezier", "target-arrow-shape": "triangle", "arrow-scale": 0.7, opacity: 0.7 } },
    { selector: ".faded", style: { opacity: 0.12 } },
    { selector: "node:selected", style: { "border-width": 3, "border-color": "#FFFFFF" } },
  ];
  for (const d of NODE_TYPE_DEFS) styles.push({ selector: `node.${d.key}`, style: { "background-color": d.color, shape: d.shape } });
  for (const d of EDGE_TYPE_DEFS) styles.push({ selector: `edge.${d.key}`, style: { "line-color": d.color, "target-arrow-color": d.color } });
  for (const level of Object.keys(riskMeta) as (keyof typeof riskMeta)[]) {
    styles.push({ selector: `node.risk-${level}`, style: { "border-color": riskMeta[level].color } });
  }
  return styles;
}
