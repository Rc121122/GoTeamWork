declare module '../bindings/GOproject/app.js' {
  export interface UserBinding {
    id: string;
    name: string;
    roomId?: string | null;
    isOnline: boolean;
  }

  export function Greet(name: string): Promise<string>;
  export function GetMode(): Promise<string>;
  export function ListAllUsers(): Promise<UserBinding[]>;
}
