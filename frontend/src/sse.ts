import { globalState } from "./state";
import type {
  ChatMessage,
  CopiedItem,
  InviteEventPayload,
  SSEEnvelope,
  User,
} from "./api/types";

const API_BASE_URL = "http://localhost:8080";

export type SSEEventType = 
  | 'user_created' 
  | 'user_offline' 
  | 'user_invited' 
  | 'user_joined' 
  | 'chat_message' 
  | 'clipboard_copied' 
  | 'clipboard_updated'
  | 'connected' 
  | 'disconnected';

type Listener<T = any> = (data: T) => void;

const listeners: Record<string, Listener[]> = {};

export function addSSEListener<T>(event: SSEEventType, callback: Listener<T>) {
    if (!listeners[event]) {
        listeners[event] = [];
    }
    listeners[event].push(callback);
}

export function removeSSEListener<T>(event: SSEEventType, callback: Listener<T>) {
    if (!listeners[event]) return;
    listeners[event] = listeners[event].filter(cb => cb !== callback);
}

function dispatch<T>(event: SSEEventType, data: T) {
    if (listeners[event]) {
        listeners[event].forEach(cb => cb(data));
    }
}

function parseEnvelope<T>(event: MessageEvent<string>): T {
  const parsed = JSON.parse(event.data) as SSEEnvelope<T>;
  return parsed.data;
}

export function connectSSE(userId: string): void {
  // If already connected, close existing? Or just return?
  // For now, let's close existing to be safe if userId changes
  if (globalState.sseConnection) {
      globalState.sseConnection.close();
  }

  const url = `${API_BASE_URL}/api/sse?userId=${encodeURIComponent(userId)}`;

  const setup = (): void => {
    const source = new EventSource(url);
    globalState.sseConnection = source;

    source.addEventListener("user_created", (event) => {
      dispatch('user_created', parseEnvelope<User>(event as MessageEvent<string>));
    });

    source.addEventListener("user_offline", (event) => {
      dispatch('user_offline', parseEnvelope<{ userId: string }>(event as MessageEvent<string>));
    });

    source.addEventListener("user_invited", (event) => {
      console.log("SSE user_invited event received:", event.data);
      const payload = parseEnvelope<InviteEventPayload>(event as MessageEvent<string>);
      dispatch('user_invited', payload);
    });

    source.addEventListener("user_joined", (event) => {
      console.log("SSE user_joined event received:", event.data);
      const payload = parseEnvelope<{ roomId: string; roomName: string; userId: string; userName: string }>(event as MessageEvent<string>);
      dispatch('user_joined', payload);
    });

    source.addEventListener("chat_message", (event) => {
      dispatch('chat_message', parseEnvelope<ChatMessage>(event as MessageEvent<string>));
    });

    source.addEventListener("clipboard_copied", (event) => {
      dispatch('clipboard_copied', parseEnvelope<CopiedItem>(event as MessageEvent<string>));
    });

    source.addEventListener("connected", () => {
      dispatch('connected', null);
    });

    source.addEventListener("heartbeat", () => {
      // Heartbeat
    });

    source.onerror = () => {
      source.close();
      globalState.sseConnection = null;
      dispatch('disconnected', null);
      window.setTimeout(setup, 5000);
    };
  };

  setup();
}
