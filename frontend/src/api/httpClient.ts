import type {
  ApiMessageResponse,
  AcceptInviteRequest,
  ChatMessage,
  ChatMessageRequest,
  CreateUserRequest,
  InviteUserRequest,
  JoinRoomRequest,
  LeaveRoomRequest,
  User,
  Room,
  Operation,
} from "./types";

let API_BASE_URL = "http://localhost:8080";

/**
 * Parse and normalize server URL for different connection types:
 * - localhost or 127.0.0.1 -> http://localhost:8080
 * - LAN IP (192.168.x.x, 10.x.x.x) -> http://IP:8080
 * - Cloudflare tunnel (https://xxx.trycloudflare.com) -> use as-is (no port)
 * - Custom URL with port -> use as-is
 */
export function parseServerUrl(input: string): string {
  let url = input.trim();
  
  // Empty or default
  if (!url || url === 'localhost') {
    return 'http://localhost:8080';
  }
  
  // Already has protocol
  if (url.startsWith('http://') || url.startsWith('https://')) {
    // HTTPS URLs (like Cloudflare tunnels) - don't add port
    if (url.startsWith('https://')) {
      return url.replace(/\/$/, ''); // Remove trailing slash only
    }
    // HTTP URL - add port if missing
    const hasPort = /:(\/\/[^/]+):\d+/.test(url) || /:\d+(\/|$)/.test(url.replace('http://', ''));
    if (!hasPort) {
      url = url.replace(/\/$/, '') + ':8080';
    }
    return url.replace(/\/$/, '');
  }
  
  // No protocol - assume HTTP and add port
  return `http://${url}:8080`.replace(/:8080:8080/, ':8080'); // Prevent double port
}

export function setApiBaseUrl(input: string) {
  API_BASE_URL = parseServerUrl(input);
  console.log("API Base URL set to:", API_BASE_URL);
}

export function getApiBaseUrl(): string {
  return API_BASE_URL;
}

export class HttpError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "HttpError";
    Object.setPrototypeOf(this, HttpError.prototype);
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      "Content-Type": "application/json",
    },
    ...init,
  });

  if (!response.ok) {
    throw new HttpError(response.status, `Request to ${path} failed with status ${response.status}`);
  }

  return (await response.json()) as T;
}

export async function httpFetchUsers(): Promise<User[]> {
  return request<User[]>("/api/users");
}

export async function httpCreateUser(payload: CreateUserRequest): Promise<User> {
  return request<User>("/api/users", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpInviteUser(payload: InviteUserRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/invite", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpAcceptInvite(payload: AcceptInviteRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/invite/accept", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpJoinRoom(payload: JoinRoomRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/join", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpRequestJoin(payload: JoinRoomRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/join/request", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpApproveJoin(ownerId: string, requesterId: string, roomId: string): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/join/approve", {
    method: "POST",
    body: JSON.stringify({ ownerId, requesterId, roomId }),
  });
}

export async function httpLeaveRoom(payload: LeaveRoomRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/leave", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpFetchRooms(): Promise<Room[]> {
  return request<Room[]>("/api/rooms");
}

export async function httpCreateRoom(name: string): Promise<Room> {
  return request<Room>("/api/rooms", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function httpFetchChatHistory(roomId: string): Promise<ChatMessage[]> {
  return request<ChatMessage[]>(`/api/chat/${roomId}`);
}

export async function httpSendChatMessage(payload: ChatMessageRequest): Promise<ApiMessageResponse> {
  return request<ApiMessageResponse>("/api/chat", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function httpFetchOperations(roomId: string, sinceId: string = ""): Promise<Operation[]> {
  let url = `/api/operations/${roomId}`;
  if (sinceId) {
    url += `?since=${sinceId}`;
  }
  return request<Operation[]>(url);
}

