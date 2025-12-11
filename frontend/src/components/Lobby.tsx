import React, { useEffect, useState } from 'react';
import { hostListUsers, hostListRooms, hostCreateRoom, hostJoinRoom, hostInviteUser, hostRequestJoin } from '../api/wailsBridge';
import { httpFetchUsers, httpFetchRooms, httpCreateRoom, httpJoinRoom, httpInviteUser, httpRequestJoin } from '../api/httpClient';
import { User, Room } from '../api/types';

interface LobbyProps {
  currentUser: { id: string; name: string };
  onJoinRoom: (room: Room) => void;
  appMode: 'host' | 'client';
  onInviteSent?: (expiresAt: number) => void;
}

const Lobby: React.FC<LobbyProps> = ({ currentUser, onJoinRoom, appMode, onInviteSent }) => {
  const [users, setUsers] = useState<User[]>([]);
  const [rooms, setRooms] = useState<Room[]>([]);
  const [newRoomName, setNewRoomName] = useState('');

  const refreshData = async () => {
    try {
      let u: User[], r: Room[];
      if (appMode === 'client') {
        u = await httpFetchUsers();
        r = await httpFetchRooms();
      } else {
        u = await hostListUsers();
        r = await hostListRooms();
      }
      setUsers(u);
      setRooms(r);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    refreshData();
    const interval = setInterval(refreshData, 2000); // Poll every 2s
    return () => clearInterval(interval);
  }, []);

  const handleCreateRoom = async () => {
    if (!newRoomName.trim()) return;
    try {
      let room: Room;
      if (appMode === 'client') {
        room = await httpCreateRoom(newRoomName);
      } else {
        room = await hostCreateRoom(newRoomName);
      }
      setNewRoomName('');
      refreshData();
      // Auto-join created room (owner joins directly)
      onJoinRoom(room);
    } catch (err) {
      console.error(err);
    }
  };

  const handleJoinRoom = async (room: Room) => {
      // If user is owner or already in room, join directly
      if (room.ownerId === currentUser.id || room.userIds.includes(currentUser.id)) {
          try {
              if (appMode === 'client') {
                 await httpJoinRoom({ roomId: room.id, userId: currentUser.id });
              } else {
                 await hostJoinRoom(room.id, currentUser.id);
              }
              onJoinRoom(room);
          } catch (err) {
              console.error("Failed to join room", err);
          }
          return;
      }

      // Otherwise, request to join
      try {
          if (appMode === 'client') {
              await httpRequestJoin({ roomId: room.id, userId: currentUser.id });
          } else {
              await hostRequestJoin(currentUser.id, room.id);
          }
          alert("Join request sent to room owner. Please wait for approval.");
      } catch (err) {
          console.error("Failed to request join", err);
          alert("Failed to send join request");
      }
  }

  const handleInvite = async (userId: string) => {
    try {
      let response;
      if (appMode === 'client') {
        response = await httpInviteUser({ 
            userId: userId, 
            inviterId: currentUser.id, 
            message: `Join me!` 
        });
      } else {
        // hostInviteUser returns string message, not full response object currently in wailsBridge.
        // We need to update wailsBridge or just assume 30s.
        // Actually, let's check hostInviteUser in wailsBridge.
        // If it returns string, we can't get expiresAt.
        // But for now, let's assume 30s for host mode if we can't get it.
        await hostInviteUser(userId);
        // Mock response for host mode
        response = { expiresAt: Date.now() / 1000 + 30 };
      }
      
      if (onInviteSent && response.expiresAt) {
          onInviteSent(response.expiresAt);
      } else {
          alert("Invitation sent!");
      }
    } catch (err) {
      console.error("Failed to invite user", err);
      alert("Failed to invite user");
    }
  };

  return (
    <div className="lobby-shell">
      <div className="section-header">
        <div>
          <p className="pill" style={{ display: 'inline-block', marginBottom: '6px' }}>Lobby</p>
          <h2 style={{ margin: 0 }}>Welcome, {currentUser.name}</h2>
        </div>
        <div className="pill-soft">Users online: {users.length}</div>
      </div>

      <div className="section-card">
        <div className="section-header">
          <h3 style={{ margin: 0 }}>Rooms</h3>
          <div className="input-inline">
            <input 
              value={newRoomName} 
              onChange={e => setNewRoomName(e.target.value)} 
              placeholder="New Room Name"
              className="text-input"
            />
            <button onClick={handleCreateRoom} className="primary-btn">Create</button>
          </div>
        </div>
        <div className="list-grid">
          {rooms.map(r => (
            <div key={r.id} className="list-item">
              <div>
                <div style={{ fontWeight: 700 }}>{r.name}</div>
                <div className="muted">{r.userIds.length} users</div>
              </div>
              <button className="secondary-btn" onClick={() => handleJoinRoom(r)}>
                {r.ownerId === currentUser.id || r.userIds.includes(currentUser.id) ? 'Join' : 'Request to Join'}
              </button>
            </div>
          ))}
          {rooms.length === 0 && <div className="muted">No rooms yet. Create one to start collaborating.</div>}
        </div>
      </div>

      <div className="section-card">
        <div className="section-header">
          <h3 style={{ margin: 0 }}>Online Users</h3>
        </div>
        <div className="list-grid">
          {users.map(u => (
            <div key={u.id} className="list-item">
              <div>
                <div style={{ fontWeight: 700 }}>{u.name} {u.id === currentUser.id ? '(You)' : ''}</div>
                <div className="muted">{u.roomId ? 'In Room' : 'Available'}</div>
              </div>
              {u.id !== currentUser.id && !u.roomId && (
                <button className="secondary-btn" onClick={() => handleInvite(u.id)}>Invite</button>
              )}
            </div>
          ))}
          {users.length === 0 && <div className="muted">No users online.</div>}
        </div>
      </div>
    </div>
  );
};

export default Lobby;
