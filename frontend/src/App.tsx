import { useState, useEffect, useCallback } from 'react';
import { getAppMode, setAppMode as setBackendMode } from './api/wailsBridge';
import { EventsOn, WindowSetPosition, WindowSetSize, WindowShow, WindowSetAlwaysOnTop, WindowCenter, WindowUnmaximise, WindowReload } from '../wailsjs/runtime/runtime';
import { GetClipboardType, SetPendingClipboardFiles, SaveDroppedFiles, ShareSystemClipboard } from '../wailsjs/go/main/App';
import HUD from './components/HUD';
import Sidebar from './components/Sidebar';
import LandingPage from './components/LandingPage';
import ModeSelect from './components/ModeSelect';
import NewUserPage from './components/NewUserPage';
import Lobby from './components/Lobby';
import RoomView from './components/Room';
import HostDashboard from './components/HostDashboard';
import TitleBar from './components/TitleBar';
import { SettingsModal, AboutModal } from './components/Modals';
import { AppState } from './types/fsm';
import { User, Room, InviteEventPayload } from './api/types';
import { connectSSE, addSSEListener, removeSSEListener } from './sse';
import { httpAcceptInvite, httpFetchRooms, httpApproveJoin } from './api/httpClient';
import { hostApproveJoin } from './api/wailsBridge';
import './app.css';

function App() {
  const [appMode, setAppModeState] = useState<'host' | 'client' | 'pending'>('pending');
  const [state, setState] = useState<AppState>('LOADING');
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [currentRoom, setCurrentRoom] = useState<Room | null>(null);
  
  const [isHUD, setIsHUD] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showAbout, setShowAbout] = useState(false);
  const [pendingInvite, setPendingInvite] = useState<InviteEventPayload | null>(null);
  const [joinRequest, setJoinRequest] = useState<{ roomId: string; roomName: string; requesterId: string; requesterName: string } | null>(null);
  
  const [inviterWaiting, setInviterWaiting] = useState(false);
  const [inviterExpiresAt, setInviterExpiresAt] = useState<number>(0);
  const [timeLeft, setTimeLeft] = useState(0);
  const [isHUDEnabled, setIsHUDEnabled] = useState(true);
  const [hudContentType, setHudContentType] = useState<string>('text');
  const [isDragOver, setIsDragOver] = useState(false);
  const [dragError, setDragError] = useState<string | null>(null);
  const [mainWindowSize, setMainWindowSize] = useState({ width: 1024, height: 768 });

  const HUD_WINDOW_WIDTH = 320;
  const HUD_WINDOW_HEIGHT = 320;
  const HUD_WINDOW_VERTICAL_OFFSET = 20;

  const showHudAtCursor = ({ screenX, screenY }: { screenX?: number; screenY?: number } = {}) => {
    const fallbackX = window.screenX + window.innerWidth / 2;
    const fallbackY = window.screenY + window.innerHeight / 2;
    const targetX = Math.max(0, Math.round((screenX ?? fallbackX) - HUD_WINDOW_WIDTH / 2));
    const targetY = Math.max(0, Math.round((screenY ?? fallbackY) - HUD_WINDOW_HEIGHT - HUD_WINDOW_VERTICAL_OFFSET));
    WindowUnmaximise();
    WindowSetSize(HUD_WINDOW_WIDTH, HUD_WINDOW_HEIGHT);
    WindowSetPosition(targetX, targetY);
    WindowShow();
    
    // Aggressively set AlwaysOnTop with multiple retries for Windows reliability
    const setAlwaysOnTopRetry = () => {
      WindowSetAlwaysOnTop(true);
      setTimeout(() => WindowSetAlwaysOnTop(true), 50);
      setTimeout(() => WindowSetAlwaysOnTop(true), 100);
      setTimeout(() => WindowSetAlwaysOnTop(true), 200);
      setTimeout(() => WindowSetAlwaysOnTop(true), 500);
    };
    setAlwaysOnTopRetry();
  };

  type DroppedFile = { file: File; rel: string };

  const walkEntry = async (entry: any, prefix: string): Promise<DroppedFile[]> => {
    return new Promise((resolve, reject) => {
      if (entry?.isFile) {
        entry.file((file: File) => resolve([{ file, rel: prefix }]), reject);
        return;
      }

      if (entry?.isDirectory) {
        const dirPrefix = prefix ? `${prefix}/${entry.name}` : entry.name;
        const reader = entry.createReader();
        const files: DroppedFile[] = [];

        const readEntries = () => {
          reader.readEntries(async (entries: any[]) => {
            if (!entries.length) {
              resolve(files);
              return;
            }

            try {
              for (const child of entries) {
                const childFiles = await walkEntry(child, dirPrefix);
                files.push(...childFiles);
              }
              readEntries();
            } catch (err) {
              reject(err);
            }
          }, reject);
        };

        readEntries();
        return;
      }

      resolve([]);
    });
  };

  const collectDroppedFiles = async (dataTransfer: DataTransfer): Promise<DroppedFile[]> => {
    const items = Array.from(dataTransfer.items || []);

    const entryPromises = items
      .map((item) => {
        const entry = (item as any).webkitGetAsEntry?.();
        if (!entry) return null;
        return walkEntry(entry, '');
      })
      .filter(Boolean) as Promise<DroppedFile[]>[];

    try {
      const entryResults = (await Promise.all(entryPromises)).flat();
      if (entryResults.length) {
        return entryResults;
      }
    } catch (err) {
      console.warn('[drop] entry traversal failed', err);
    }

    const fallbackFiles = items
      .map(i => i.kind === 'file' ? i.getAsFile() : null)
      .filter((f): f is File => Boolean(f));
    const files = fallbackFiles.length ? fallbackFiles : Array.from(dataTransfer.files || []);

    return files.map((file) => {
      const relPathRaw = (file as any).webkitRelativePath || '';
      const rel = relPathRaw ? relPathRaw.replace(/^[/\\]+/, '').split('/').slice(0, -1).join('/') : '';
      return { file, rel };
    });
  };

  // Timer effect
  useEffect(() => {
      const timer = setInterval(() => {
          const now = Date.now() / 1000;
          if (inviterWaiting && inviterExpiresAt > now) {
              setTimeLeft(Math.ceil(inviterExpiresAt - now));
          } else if (pendingInvite && pendingInvite.expiresAt > now) {
              setTimeLeft(Math.ceil(pendingInvite.expiresAt - now));
          } else {
              setTimeLeft(0);
              if (inviterWaiting) setInviterWaiting(false);
              if (pendingInvite) setPendingInvite(null);
          }
      }, 1000);
      return () => clearInterval(timer);
  }, [inviterWaiting, inviterExpiresAt, pendingInvite]);

  const handleInviteSent = (expiresAt: number) => {
      setInviterExpiresAt(expiresAt);
      setInviterWaiting(true);
  };

  const fetchAndJoinRoom = useCallback(async (roomId: string) => {
    try {
        const rooms = await httpFetchRooms();
        const room = rooms.find(r => r.id === roomId);
        if (room) {
            console.log("Joining room:", room);
            setCurrentRoom(room);
            setState('ROOM');
        }
    } catch (e) {
        console.error("Failed to fetch room details", e);
    }
  }, []);

  useEffect(() => {
    getAppMode().then((mode) => {
      if (mode === 'host') {
        setAppModeState('host');
        setState('HOST_DASHBOARD');
      } else if (mode === 'client') {
        setAppModeState('client');
        setState('LANDING');
      } else {
        setAppModeState('pending');
        setState('MODE_SELECT');
      }
    }).catch((err) => {
        console.error("Failed to get app mode", err);
        setAppModeState('pending');
        setState('MODE_SELECT');
    });
  }, []);

  useEffect(() => {
    // Listen for HUD trigger
    const cancelListener = EventsOn("clipboard:show-share-button", async (data: { screenX: number, screenY: number }) => {
        if (appMode === 'host') {
            console.log("HUD ignored in host mode");
            return;
        }
        if (!isHUDEnabled) {
            console.log("HUD disabled by user");
            return;
        }

        console.log("HUD Triggered", data);
        
        // Get clipboard content type
        try {
            const contentType = await GetClipboardType();
            console.log("Clipboard type:", contentType);
            setHudContentType(contentType);
        } catch (err) {
            console.error("Failed to get clipboard type:", err);
            setHudContentType('text'); // fallback
        }
        
        setIsHUD(true);
        showHudAtCursor(data);
    });

    return () => {
        if (cancelListener) cancelListener();
    };
  }, [appMode, isHUDEnabled]);

  // Global drag-and-drop for sharing files/folders into the app
  useEffect(() => {
    const handleDragOver = (event: DragEvent) => {
      if (!event.dataTransfer) return;
      if (Array.from(event.dataTransfer.types).includes('Files')) {
        event.preventDefault();
        event.dataTransfer.dropEffect = 'copy';
        event.dataTransfer.effectAllowed = 'copy';
        setIsDragOver(true);
        console.debug('[dragover] Files detected; dropEffect set to copy');
      }
    };

    const handleDragLeave = (event: DragEvent) => {
      // Only reset when leaving document
      if ((event.target as HTMLElement)?.contains?.(document.body)) return;
      setIsDragOver(false);
    };

    const handleDrop = async (event: DragEvent) => {
      if (!event.dataTransfer) return;
      event.preventDefault();
      setIsDragOver(false);
      setDragError(null);

      // Collect files (recurses folders on supporting browsers)
      const dtItems = Array.from(event.dataTransfer.items || []);
      const droppedFiles = await collectDroppedFiles(event.dataTransfer);
      const fileList = droppedFiles.map(df => df.file);

      // Debug: enumerate types & items
      const types = Array.from(event.dataTransfer.types || []);
      const items = Array.from(event.dataTransfer.items || []);
      console.debug('[drop] types', types);
      console.debug('[drop] items', items.map(i => ({ kind: i.kind, type: i.type })));

      const publicFileUrl = event.dataTransfer.getData('public.file-url');
      if (publicFileUrl) {
        console.debug('[drop] public.file-url raw', publicFileUrl);
      }

      const nsFiles = event.dataTransfer.getData('com.apple.nsfilenames');
      if (nsFiles) {
        console.debug('[drop] com.apple.nsfilenames raw', nsFiles);
      }

      // Try file:// URI list first (Finder usually provides this)
      const uriListRaw = event.dataTransfer.getData('text/uri-list');
      const uriPaths = uriListRaw
        ? uriListRaw
            .split(/\r?\n/)
            .map(line => line.trim())
            .filter(line => line.startsWith('file://'))
            .map(line => decodeURI(line.replace('file://', '')))
        : [];

      // Try explicit file paths from File objects (may be empty in WKWebView)
      const filePaths = fileList.map(f => (f as any).path).filter(Boolean) as string[];

      // Try getAsString on string items (public.file-url) which can work on WKWebView
      const stringPathPromises = dtItems
        .filter(i => i.kind === 'string')
        .map(i => new Promise<string>((resolve) => {
          i.getAsString((s) => resolve(s || ''));
        }));
      const stringPathResults = await Promise.all(stringPathPromises);
      const stringPaths = stringPathResults
        .flatMap(raw => raw.split(/\r?\n/))
        .map(line => line.trim())
        .filter(line => line.startsWith('file://'))
        .map(line => decodeURI(line.replace('file://', '')));

      // Merge unique paths
      const paths = Array.from(new Set([...uriPaths, ...filePaths, ...stringPaths]));

      console.debug('[drop] files length', fileList.length, 'uriPaths', uriPaths, 'filePaths length', filePaths.length, 'merged paths length', paths.length);
      if (fileList.length && !paths.length) {
        console.warn('[drop] files present but no paths resolved (webview sandbox may hide paths). URI list raw:', uriListRaw);
      }

      let effectivePaths = paths;

      // Fallback: if no paths, pull file bytes into temp files via backend
      if (!effectivePaths.length && fileList.length) {
        try {
          console.debug('[drop] attempting in-memory transfer of files');
          const payloads = await Promise.all(
            droppedFiles.map(async ({ file, rel }) => {
              const base64 = await readFileToBase64(file);
              return { name: file.name || 'dropped.bin', rel, data: base64 };
            })
          );
          effectivePaths = await SaveDroppedFiles(payloads as any);
          console.debug('[drop] in-memory transfer produced paths', effectivePaths);
        } catch (err) {
          console.error('[drop] in-memory transfer failed', err);
        }
      }

      if (!effectivePaths.length) {
        console.warn('[drop] no usable file paths; aborting');
        return;
      }

      try {
        await SetPendingClipboardFiles(effectivePaths);
        console.debug('[drop] cached paths, invoking ShareSystemClipboard');
        await ShareSystemClipboard();
        console.debug('[drop] share complete');
      } catch (err) {
        console.error('Failed to share dropped files', err);
        setDragError('Failed to share dropped files');
      }
    };

    window.addEventListener('dragover', handleDragOver);
    window.addEventListener('dragleave', handleDragLeave);
    window.addEventListener('drop', handleDrop);

    return () => {
      window.removeEventListener('dragover', handleDragOver);
      window.removeEventListener('dragleave', handleDragLeave);
      window.removeEventListener('drop', handleDrop);
    };
  }, []);

  // Convert File to base64 safely (arrayBuffer with fallback to FileReader)
  const readFileToBase64 = (file: File): Promise<string> => {
    const arrayBufferToBase64 = (buffer: ArrayBuffer): string => {
      const bytes = new Uint8Array(buffer);
      const chunkSize = 0x8000;
      let binary = '';
      for (let i = 0; i < bytes.length; i += chunkSize) {
        const chunk = bytes.subarray(i, i + chunkSize);
        binary += String.fromCharCode.apply(null, Array.from(chunk));
      }
      return btoa(binary);
    };

    return file.arrayBuffer()
      .then(arrayBufferToBase64)
      .catch((err) => {
        console.warn('[drop] arrayBuffer failed, falling back to FileReader', err);
        return new Promise((resolve, reject) => {
          const reader = new FileReader();
          reader.onerror = () => reject(reader.error);
          reader.onload = () => {
            try {
              const result = reader.result as string;
              const base64 = result.startsWith('data:') ? result.split(',')[1] : result;
              resolve(base64);
            } catch (e) {
              reject(e);
            }
          };
          reader.readAsDataURL(file);
        });
      });
  };

  // Connect to SSE when currentUser is set
  useEffect(() => {
    if (currentUser) {
      console.log("Connecting to SSE for user:", currentUser.id);
      connectSSE(currentUser.id);

      const onInvite = (payload: InviteEventPayload) => {
          console.log("Received invite:", payload);
          setPendingInvite(payload);
      };

      const onJoin = (payload: { roomId: string; roomName: string; userId: string; userName: string }) => {
          console.log("User joined room:", payload);
          if (payload.userId === currentUser.id) {
             void fetchAndJoinRoom(payload.roomId);
          }
          setInviterWaiting(false);
      };

      const onJoinRequest = (payload: { roomId: string; roomName: string; requesterId: string; requesterName: string }) => {
          console.log("Received join request:", payload);
          setJoinRequest(payload);
      };

      const onDisconnect = () => {
          console.warn("SSE Disconnected");
      };

      addSSEListener('user_invited', onInvite);
      addSSEListener('user_joined', onJoin);
      addSSEListener('join_request', onJoinRequest);
      addSSEListener('disconnected', onDisconnect);

      return () => {
          removeSSEListener('user_invited', onInvite);
          removeSSEListener('user_joined', onJoin);
          removeSSEListener('join_request', onJoinRequest);
          removeSSEListener('disconnected', onDisconnect);
      };
    }
  }, [appMode, currentUser, fetchAndJoinRoom]);

  const handleApproveJoinRequest = async () => {
      if (!joinRequest || !currentUser) return;
      try {
          if (appMode === 'client') {
              await httpApproveJoin(currentUser.id, joinRequest.requesterId, joinRequest.roomId);
          } else {
              await hostApproveJoin(currentUser.id, joinRequest.requesterId, joinRequest.roomId);
          }
          setJoinRequest(null);
      } catch (err) {
          console.error("Failed to approve join request", err);
          alert("Failed to approve request");
      }
  };

  const handleRejectJoinRequest = () => {
      setJoinRequest(null);
  };

    const closeHUD = () => {
      setIsHUD(false);
      WindowSetAlwaysOnTop(false);
      // Restore main window size
      WindowSetSize(mainWindowSize.width, mainWindowSize.height);
      WindowCenter();
    };

  const handleReboot = () => {
    WindowReload();
  };

  // FSM Transitions
  const handleStart = () => {
    setState('NEW_USER');
  };

  const handleSelectMode = async (mode: 'host' | 'client') => {
    try {
      await setBackendMode(mode);
      setAppModeState(mode);
      if (mode === 'host') {
        setState('HOST_DASHBOARD');
      } else {
        setState('LANDING');
      }
    } catch (err) {
      console.error('Failed to set app mode', err);
    }
  };

  const handleUserCreated = (user: { id: string; name: string }) => {
    setCurrentUser({ ...user, roomId: null, isOnline: true });
    setState('LOBBY');
  };

  const handleJoinRoom = (room: Room) => {
    setCurrentRoom(room);
    setState('ROOM');
  };

  const handleLeaveRoom = () => {
    setCurrentRoom(null);
    setState('LOBBY');
  };

  const handleAcceptInvite = async () => {
    if (!pendingInvite || !currentUser) return;
    try {
      const response = await httpAcceptInvite({
        inviteId: pendingInvite.inviteId,
        inviteeId: currentUser.id
      });
      setPendingInvite(null);
      
      if (response.roomId) {
          void fetchAndJoinRoom(response.roomId);
      }
    } catch (err) {
      console.error("Failed to accept invite", err);
      alert("Failed to accept invite");
    }
  };

  const handleDeclineInvite = () => {
    setPendingInvite(null);
  };

  if (state === 'LOADING') {
    return <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh'}}>Loading...</div>;
  }

  // When HUD is active, render ONLY the HUD with solid background
  if (isHUD) {
    return <HUD onClose={closeHUD} contentType={hudContentType} />;
  }

  if (state === 'MODE_SELECT') {
    return (
      <div className="app-container">
        <TitleBar />
        <ModeSelect onSelect={handleSelectMode} />
      </div>
    );
  }

  return (
    <div className="app-container">
      {isDragOver && (
        <div className="drag-overlay">
          <div className="drag-overlay__card">
            <div className="drag-overlay__title">Drop to share</div>
            <div className="drag-overlay__hint">Files and folders are supported</div>
            {dragError && <div className="drag-overlay__error">{dragError}</div>}
          </div>
        </div>
      )}
      <TitleBar />
      <div className="content">
        <Sidebar 
            onReboot={handleReboot}
            onSettings={() => setShowSettings(true)}
            onAbout={() => setShowAbout(true)}
        />
        
        <main className="main-area">
            {state === 'LANDING' && <LandingPage onStart={handleStart} />}
            {state === 'NEW_USER' && <NewUserPage onUserCreated={handleUserCreated} appMode={appMode} />}
            {state === 'LOBBY' && currentUser && <Lobby currentUser={currentUser} onJoinRoom={handleJoinRoom} appMode={appMode} onInviteSent={handleInviteSent} />}
            {state === 'ROOM' && currentUser && currentRoom && <RoomView currentUser={currentUser} currentRoom={currentRoom} onLeave={handleLeaveRoom} appMode={appMode} />}
            {state === 'HOST_DASHBOARD' && <HostDashboard />}
        </main>
      </div>

      <SettingsModal 
          isOpen={showSettings} 
          onClose={() => setShowSettings(false)} 
          isHUDEnabled={isHUDEnabled}
          onToggleHUD={() => setIsHUDEnabled(!isHUDEnabled)}
      />
      <AboutModal isOpen={showAbout} onClose={() => setShowAbout(false)} />
      {/* Inviter Waiting Modal */}
      {inviterWaiting && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.7)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 2000
        }}>
          <div style={{
            background: '#2c3e50', padding: '30px', borderRadius: '10px', width: '400px', color: 'white',
            boxShadow: '0 10px 25px rgba(0,0,0,0.5)', textAlign: 'center'
          }}>
            <h3 style={{ marginTop: 0 }}>Invitation Sent</h3>
            <p style={{ fontSize: '1.1rem', margin: '20px 0' }}>
              Waiting for user to accept...
            </p>
            <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#3498db', marginBottom: '20px' }}>
              {timeLeft}s
            </div>
            <button 
              onClick={() => setInviterWaiting(false)}
              style={{ padding: '10px 25px', background: '#95a5a6', border: 'none', borderRadius: '5px', color: 'white', cursor: 'pointer', fontSize: '1rem' }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Invite Modal */}
      {pendingInvite && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.7)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 2000
        }}>
          <div style={{
            background: '#2c3e50', padding: '30px', borderRadius: '10px', width: '400px', color: 'white',
            boxShadow: '0 10px 25px rgba(0,0,0,0.5)', textAlign: 'center'
          }}>
            <h3 style={{ marginTop: 0 }}>Invitation Received!</h3>
            <p style={{ fontSize: '1.1rem', margin: '20px 0' }}>
              <strong>{pendingInvite.inviter}</strong> invited you to join a room.
            </p>
            <p style={{ fontStyle: 'italic', color: '#bdc3c7', marginBottom: '20px' }}>
              "{pendingInvite.message}"
            </p>
            <div style={{ fontSize: '1.5rem', fontWeight: 'bold', color: '#e74c3c', marginBottom: '20px' }}>
              {timeLeft}s
            </div>
            <div style={{ display: 'flex', justifyContent: 'center', gap: '20px' }}>
              <button 
                onClick={handleAcceptInvite}
                style={{ padding: '10px 25px', background: '#27ae60', border: 'none', borderRadius: '5px', color: 'white', cursor: 'pointer', fontSize: '1rem' }}
              >
                Accept
              </button>
              <button 
                onClick={handleDeclineInvite}
                style={{ padding: '10px 25px', background: '#c0392b', border: 'none', borderRadius: '5px', color: 'white', cursor: 'pointer', fontSize: '1rem' }}
              >
                Decline
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Join Request Modal */}
      {joinRequest && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.7)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 2000
        }}>
          <div style={{
            background: '#2c3e50', padding: '30px', borderRadius: '10px', width: '400px', color: 'white',
            boxShadow: '0 10px 25px rgba(0,0,0,0.5)', textAlign: 'center'
          }}>
            <h3 style={{ marginTop: 0 }}>Join Request</h3>
            <p style={{ fontSize: '1.1rem', margin: '20px 0' }}>
              <strong>{joinRequest.requesterName}</strong> wants to join your room <strong>{joinRequest.roomName}</strong>.
            </p>
            <div style={{ display: 'flex', justifyContent: 'center', gap: '20px' }}>
              <button 
                onClick={handleApproveJoinRequest}
                style={{ padding: '10px 25px', background: '#27ae60', border: 'none', borderRadius: '5px', color: 'white', cursor: 'pointer', fontSize: '1rem' }}
              >
                Approve
              </button>
              <button 
                onClick={handleRejectJoinRequest}
                style={{ padding: '10px 25px', background: '#c0392b', border: 'none', borderRadius: '5px', color: 'white', cursor: 'pointer', fontSize: '1rem' }}
              >
                Reject
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
