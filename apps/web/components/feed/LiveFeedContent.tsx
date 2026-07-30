"use client";
import { useState } from "react";
import { useEvents } from "@/lib/hooks/queries";
import { useLiveEvents } from "@/lib/hooks/live";
import { filterEvents } from "@/lib/feed/filter";
import { EMPTY_FILTERS } from "@/lib/api/types";
import type { FeedFilters as Filters, FeedEvent } from "@/lib/api/types";
import { FeedFilters } from "./FeedFilters";
import { FeedTable } from "./FeedTable";
import { EventDetailDrawer } from "./EventDetailDrawer";

export function LiveFeedContent() {
  useLiveEvents();
  const { data } = useEvents();
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS);
  const [selected, setSelected] = useState<FeedEvent | null>(null);
  const events = filterEvents(data ?? [], filters);
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <span className="relative flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-positive opacity-60" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-positive" />
        </span>
        <h1>Canlı Akış</h1>
        <span className="text-muted-foreground" style={{ fontSize: 12 }}>· {events.length} event</span>
      </div>
      <FeedFilters value={filters} onChange={setFilters} />
      <FeedTable events={events} onRowClick={setSelected} />
      <EventDetailDrawer event={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
