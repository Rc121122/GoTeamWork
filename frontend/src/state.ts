import type { User } from "./api/types";

export interface PendingInviteState {
  inviteId: string;
  inviteeId: string;
  expiresAt: number;
  countdownTimerId?: number;
}

export interface GlobalState {
  currentUser: User | null;
  isProcessingAction: boolean;
  sseConnection: EventSource | null;
  pendingInvite: PendingInviteState | null;
}

export const globalState: GlobalState = {
  currentUser: null,
  isProcessingAction: false,
  sseConnection: null,
  pendingInvite: null,
};

export function cleanup(): void {
  if (globalState.sseConnection) {
    globalState.sseConnection.close();
    globalState.sseConnection = null;
  }
  globalState.isProcessingAction = false;
    if (globalState.pendingInvite?.countdownTimerId) {
      window.clearInterval(globalState.pendingInvite.countdownTimerId);
    }
    globalState.pendingInvite = null;
}
