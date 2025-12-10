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
import { connectSSE } from './sse';
import { httpAcceptInvite, httpFetchRooms } from './api/httpClient';
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

    // Listen for HUD trigger
    const cancelListener = EventsOn("clipboard:show-share-button", (data: { screenX: number, screenY: number }) => {
        console.log("HUD Triggered", data);
        setIsHUD(true);
        // Resize window to small square
        WindowUnmaximise(); 
        WindowSetSize(150, 150);
        WindowSetPosition(data.screenX, data.screenY);
        WindowSetAlwaysOnTop(true);
        WindowShow();
    });

    return () => {
        if (cancelListener) cancelListener();
    };
  }, []);

  // Connect to SSE when currentUser is set (and mode is client)
  useEffect(() => {
    if (appMode === 'client' && currentUser) {
      console.log("Connecting to SSE for user:", currentUser.id);
      connectSSE(currentUser.id, {
        onUserInvited: (payload) => {
          console.log("Received invite:", payload);
          setPendingInvite(payload);
        },
        onUserJoined: (payload) => {
          console.log("User joined room:", payload);
          if (payload.userId === currentUser.id) {
             void fetchAndJoinRoom(payload.roomId);
          }
        },
        onDisconnected: () => {
          console.warn("SSE Disconnected");
        }
      });
    }
  }, [appMode, currentUser, fetchAndJoinRoom]);

  const closeHUD = () => {
      setIsHUD(false);
      WindowSetAlwaysOnTop(false);
      WindowSetSize(1024, 768); // Restore default size
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

  if (isHUD) {
      return <HUD onClose={closeHUD} />;
  }

  if (state === 'LOADING') {
    return <div style={{display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh'}}>Loading...</div>;
  }

  return (
    <div className="app-container" style={{display: 'flex', flexDirection: 'column', height: '100vh', backgroundColor: 'rgba(27, 38, 54, 1)', color: 'white'}}>
      <TitleBar />
      <div style={{ display: 'flex', flex: 1, overflow: 'hidden' }}>
        <Sidebar 
            onReboot={handleReboot}
            onSettings={() => setShowSettings(true)}
            onAbout={() => setShowAbout(true)}
        />
        
        <main className="content" style={{flex: 1, padding: '0', overflow: 'hidden'}}>
            {state === 'LANDING' && <LandingPage onStart={handleStart} />}
            {state === 'NEW_USER' && <NewUserPage onUserCreated={handleUserCreated} appMode={appMode} />}
            {state === 'LOBBY' && currentUser && <Lobby currentUser={currentUser} onJoinRoom={handleJoinRoom} appMode={appMode} />}
            {state === 'ROOM' && currentUser && currentRoom && <RoomView currentUser={currentUser} currentRoom={currentRoom} onLeave={handleLeaveRoom} appMode={appMode} />}
            {state === 'HOST_DASHBOARD' && <HostDashboard />}
        </main>
      </div>

      <SettingsModal isOpen={showSettings} onClose={() => setShowSettings(false)} />
      <AboutModal isOpen={showAbout} onClose={() => setShowAbout(false)} />
      
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
            <p style={{ fontStyle: 'italic', color: '#bdc3c7', marginBottom: '30px' }}>
              "{pendingInvite.message}"
            </p>
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
    </div>
  );
}

export default App;
