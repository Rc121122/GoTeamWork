import type { User } from "./api/types";

export interface GlobalState {
  currentUser: User | null;
  isProcessingAction: boolean;
  sseConnection: EventSource | null;
}

export const globalState: GlobalState = {
  currentUser: null,
  isProcessingAction: false,
  sseConnection: null,
};

export function cleanup(): void {
  if (globalState.sseConnection) {
    globalState.sseConnection.close();
    globalState.sseConnection = null;
  }
  globalState.isProcessingAction = false;
}
