// WebSocket transport: NEXT_PUBLIC_API_BASE_URL'den wss türetir, topic'e abone olur.
export function wsBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_BASE_URL;
  if (!base) throw new Error("NEXT_PUBLIC_API_BASE_URL is not set");
  return base.replace(/^http/, "ws").replace(/\/$/, "") + "/ws";
}

interface Envelope<T> { topic: string; payload: T; }

const INITIAL_BACKOFF_MS = 1000;
const MAX_BACKOFF_MS = 30000;

// wsSubscribe, verilen topic mesajlarında cb'yi çağırır. Soket kopunca (onclose/onerror)
// exponential backoff ile (1s → 2s → ... → 30s tavan) otomatik yeniden bağlanır ve aynı
// topic'e yeniden abone olur. Dönen fonksiyon aboneliği kalıcı olarak kapatır: bekleyen
// reconnect timer'ı iptal eder ve bir daha yeniden bağlanmaz.
export function wsSubscribe<T>(topic: "events" | "tokens", cb: (payload: T) => void): () => void {
  let stopped = false;
  let socket: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let backoff = INITIAL_BACKOFF_MS;

  const connect = () => {
    if (stopped) return;
    const ws = new WebSocket(wsBaseUrl());
    socket = ws;

    ws.onopen = () => {
      backoff = INITIAL_BACKOFF_MS; // sağlıklı bağlantı → backoff sıfırla
    };

    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data) as Envelope<T>;
        if (msg.topic === topic) cb(msg.payload);
      } catch {
        /* bozuk mesaj yoksay */
      }
    };

    const scheduleReconnect = () => {
      if (stopped) return;
      if (reconnectTimer) return; // zaten planlanmış
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        backoff = Math.min(backoff * 2, MAX_BACKOFF_MS);
        connect();
      }, backoff);
    };

    ws.onclose = () => {
      if (stopped) return;
      scheduleReconnect();
    };
    ws.onerror = () => {
      if (stopped) return;
      scheduleReconnect();
    };
  };

  connect();

  return () => {
    stopped = true;
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    socket?.close();
  };
}
