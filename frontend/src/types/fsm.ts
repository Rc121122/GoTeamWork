export type AppState = 
  | 'LOADING'
  | 'LANDING'
  | 'NEW_USER'
  | 'LOBBY'
  | 'ROOM'
  | 'HOST_DASHBOARD';

export interface AppContext {
  appMode: 'host' | 'client' | 'loading';
  currentUser: { id: string; name: string } | null;
  currentRoom: { id: string; name: string } | null;
}
