// WebSocket transport: NEXT_PUBLIC_API_BASE_URL'den wss türetir, topic'e abone olur.
export function wsBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) throw new Error("NEXT_PUBLIC_API_BASE_URL is not set");
  return base.replace(/^http/, "ws").replace(/\/$/, "") + "/ws";
}

interface Envelope<T> { topic: string; payload: T; }

// wsSubscribe, verilen topic mesajlarında cb'yi çağırır. Dönen fonksiyon aboneliği kapatır.
export function wsSubscribe<T>(topic: "events" | "tokens", cb: (payload: T) => void): () => void {
  const ws = new WebSocket(wsBaseUrl());
  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data) as Envelope<T>;
      if (msg.topic === topic) cb(msg.payload);
    } catch {
      /* bozuk mesaj yoksay */
    }
  };
  return () => ws.close();
}
