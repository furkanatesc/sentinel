import { describe, it, expect, vi, beforeEach } from "vitest";
import { wsSubscribe, wsBaseUrl } from "./ws";

class MockWS {
  static instances: MockWS[] = [];
  onmessage: ((e: { data: string }) => void) | null = null;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  sent: string[] = [];
  constructor(public url: string) { MockWS.instances.push(this); }
  send(d: string) { this.sent.push(d); }
  close() { this.closed = true; }
}

beforeEach(() => {
  MockWS.instances = [];
  vi.stubGlobal("WebSocket", MockWS as unknown as typeof WebSocket);
  vi.stubEnv("NEXT_PUBLIC_API_BASE_URL", "https://api.example.com");
});

describe("wsBaseUrl", () => {
  it("https → wss", () => {
    expect(wsBaseUrl()).toBe("wss://api.example.com/ws");
  });
});

describe("wsSubscribe", () => {
  it("topic mesajında cb çağrılır, unsubscribe socket kapatır", () => {
    const cb = vi.fn();
    const unsub = wsSubscribe<{ id: string }>("events", cb);
    const ws = MockWS.instances[0];
    ws.onopen?.();
    ws.onmessage?.({ data: JSON.stringify({ topic: "events", payload: { id: "e1" } }) });
    expect(cb).toHaveBeenCalledWith({ id: "e1" });
    // farklı topic → cb çağrılmaz
    ws.onmessage?.({ data: JSON.stringify({ topic: "tokens", payload: { id: "t" } }) });
    expect(cb).toHaveBeenCalledTimes(1);
    unsub();
    expect(ws.closed).toBe(true);
  });

  it("bozuk JSON mesajını yoksayar (throw etmez)", () => {
    const cb = vi.fn();
    wsSubscribe<{ id: string }>("events", cb);
    const ws = MockWS.instances[0];
    expect(() => ws.onmessage?.({ data: "not-json" })).not.toThrow();
    expect(cb).not.toHaveBeenCalled();
  });

  it("onclose sonrası backoff ile yeniden bağlanır (reconnect) ve aynı topic'e abone olur", () => {
    vi.useFakeTimers();
    try {
      const cb = vi.fn();
      wsSubscribe<{ id: string }>("events", cb);
      expect(MockWS.instances).toHaveLength(1);
      const first = MockWS.instances[0];

      // bağlantı kopar
      first.onclose?.();
      // backoff (1s) dolmadan yeni socket açılmaz
      expect(MockWS.instances).toHaveLength(1);
      vi.advanceTimersByTime(1000);
      expect(MockWS.instances).toHaveLength(2);

      // yeni socket aynı topic'e abone — mesaj cb'ye ulaşır
      const second = MockWS.instances[1];
      second.onmessage?.({ data: JSON.stringify({ topic: "events", payload: { id: "e2" } }) });
      expect(cb).toHaveBeenCalledWith({ id: "e2" });
    } finally {
      vi.useRealTimers();
    }
  });

  it("unsubscribe sonrası onclose reconnect TETİKLEMEZ (yeni socket açılmaz, timer temizlenir)", () => {
    vi.useFakeTimers();
    try {
      const cb = vi.fn();
      const unsub = wsSubscribe<{ id: string }>("events", cb);
      const first = MockWS.instances[0];

      unsub();
      expect(first.closed).toBe(true);

      // caller aboneliği kapattıktan SONRA soket onclose tetiklerse (gerçek WS'te olabilir)
      first.onclose?.();
      vi.advanceTimersByTime(60000); // en büyük backoff'u (30s) fazlasıyla aş
      expect(MockWS.instances).toHaveLength(1); // yeni socket açılmadı
    } finally {
      vi.useRealTimers();
    }
  });
});
