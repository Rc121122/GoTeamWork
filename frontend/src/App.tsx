import { useState, useEffect, useCallback } from 'react';
import { getAppMode } from './api/wailsBridge';
import { EventsOn, WindowSetPosition, WindowSetSize, WindowShow, WindowSetAlwaysOnTop, WindowCenter, WindowUnmaximise, WindowReload } from '../wailsjs/runtime/runtime';
import HUD from './components/HUD';
import Sidebar from './components/Sidebar';
import LandingPage from './components/LandingPage';
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
  const [appMode, setAppMode] = useState<'host' | 'client'>('client');
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
  const [mainWindowSize, setMainWindowSize] = useState({ width: 1024, height: 768 });

  const HUD_WINDOW_WIDTH = 160;
  const HUD_WINDOW_HEIGHT = 160;
  const HUD_WINDOW_VERTICAL_OFFSET = 20;

  const showHudAtCursor = ({ screenX, screenY }: { screenX: number; screenY: number }) => {
    const targetX = Math.max(0, Math.round(screenX - HUD_WINDOW_WIDTH / 2));
    const targetY = Math.max(0, Math.round(screenY - HUD_WINDOW_HEIGHT - HUD_WINDOW_VERTICAL_OFFSET));
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
      setAppMode(mode as 'host' | 'client');
      if (mode === 'host') {
        setState('HOST_DASHBOARD');
      } else {
        setState('LANDING');
      }
    }).catch((err) => {
        console.error("Failed to get app mode", err);
        setAppMode('client');
        setState('LANDING');
    });
  }, []);

  useEffect(() => {
    // Listen for HUD trigger
    const cancelListener = EventsOn("clipboard:show-share-button", (data: { screenX: number, screenY: number }) => {
        if (appMode === 'host') {
            console.log("HUD ignored in host mode");
            return;
        }
        if (!isHUDEnabled) {
            console.log("HUD disabled by user");
            return;
        }

        console.log("HUD Triggered", data);
        setIsHUD(true);
        showHudAtCursor(data);
    });

    return () => {
        if (cancelListener) cancelListener();
    };
  }, [appMode, isHUDEnabled]);

  // Connect to SSE when currentUser is set (and mode is client)
  useEffect(() => {
    if (appMode === 'client' && currentUser) {
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
    return <HUD onClose={closeHUD} />;
  }

  return (
    <div className="app-container">
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
