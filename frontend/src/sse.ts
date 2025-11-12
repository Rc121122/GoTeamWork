import { globalState } from "./state";
import type {
  ChatMessage,
  CopiedItem,
  InviteEventPayload,
  SSEEnvelope,
  User,
} from "./api/types";

const API_BASE_URL = "http://localhost:8080";

interface SSEHandlers {
  onUserCreated?: (user: User) => void;
  onUserInvited?: (payload: InviteEventPayload) => void;
  onChatMessage?: (message: ChatMessage) => void;
  onClipboardCopied?: (item: CopiedItem) => void;
  onDisconnected?: () => void;
}

function parseEnvelope<T>(event: MessageEvent<string>): T {
  const parsed = JSON.parse(event.data) as SSEEnvelope<T>;
  return parsed.data;
}

export function connectSSE(userId: string, handlers: SSEHandlers): void {
  const url = `${API_BASE_URL}/api/sse?userId=${encodeURIComponent(userId)}`;

  const setup = (): void => {
    const source = new EventSource(url);
    globalState.sseConnection = source;

    source.addEventListener("user_created", (event) => {
      handlers.onUserCreated?.(parseEnvelope<User>(event as MessageEvent<string>));
    });

    source.addEventListener("user_invited", (event) => {
      handlers.onUserInvited?.(parseEnvelope<InviteEventPayload>(event as MessageEvent<string>));
    });

    source.addEventListener("chat_message", (event) => {
      handlers.onChatMessage?.(parseEnvelope<ChatMessage>(event as MessageEvent<string>));
    });

    source.addEventListener("clipboard_copied", (event) => {
      handlers.onClipboardCopied?.(parseEnvelope<CopiedItem>(event as MessageEvent<string>));
    });

    source.addEventListener("connected", () => {
      // Swallow connected confirmation; console logging available if needed.
    });

    source.addEventListener("heartbeat", () => {
      // Heartbeat ensures the connection stays active.
    });

    source.onerror = () => {
      source.close();
      globalState.sseConnection = null;
      handlers.onDisconnected?.();
      window.setTimeout(setup, 5000);
    };
  };

  setup();
}
