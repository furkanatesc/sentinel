import type { ResearchSuggestion } from "@/lib/api/types";
import { Button } from "@/components/ui/button";

export function SuggestionChips({
  suggestions, onPick, disabled,
}: { suggestions: ResearchSuggestion[]; onPick: (text: string) => void; disabled: boolean }) {
  return (
    <div className="flex flex-wrap gap-2">
      {suggestions.map((s) => (
        <Button key={s.id} variant="outline" size="sm" disabled={disabled}
          className="h-auto whitespace-normal py-1.5 text-left text-xs" onClick={() => onPick(s.text)}>
          {s.text}
        </Button>
      ))}
    </div>
  );
}
