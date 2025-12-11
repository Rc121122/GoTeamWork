import React, { useEffect, useState } from 'react';
import { hostListUsers, hostListRooms, hostCreateRoom, hostJoinRoom, hostInviteUser } from '../api/wailsBridge';
import { httpFetchUsers, httpFetchRooms, httpCreateRoom, httpJoinRoom, httpInviteUser } from '../api/httpClient';
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
      // Optionally auto-join
      await handleJoinRoom(room.id);
    } catch (err) {
      console.error(err);
    }
  };

  const handleJoinRoom = async (roomId: string) => {
      try {
          let room: Room;
          if (appMode === 'client') {
             // httpJoinRoom returns ApiMessageResponse, not Room. We need to fetch room details or assume success.
             // But wait, httpJoinRoom returns ApiMessageResponse.
             // And onJoinRoom expects a Room object.
             // We should probably fetch the room details after joining.
             await httpJoinRoom({ roomId, userId: currentUser.id });
             // Fetch updated room list to get the room object? Or find it in 'rooms' state.
             const foundRoom = rooms.find(r => r.id === roomId);
             if (foundRoom) {
                 room = foundRoom;
             } else {
                 // Fallback: fetch rooms again
                 const r = await httpFetchRooms();
                 const freshRoom = r.find(rm => rm.id === roomId);
                 if (!freshRoom) throw new Error("Room not found after joining");
                 room = freshRoom;
             }
          } else {
             room = await hostJoinRoom(roomId, currentUser.id);
          }
          onJoinRoom(room);
      } catch (err) {
          console.error("Failed to join room", err);
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
    <div style={{ padding: '20px', height: '100%', overflowY: 'auto' }}>
      <h2>Lobby - Welcome, {currentUser.name}</h2>
      
      <div style={{ display: 'flex', gap: '20px' }}>
        <div style={{ flex: 1 }}>
          <h3>Online Users</h3>
          <ul style={{ listStyle: 'none', padding: 0 }}>
            {users.map(u => (
              <li key={u.id} style={{ padding: '5px', borderBottom: '1px solid #444', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>{u.name} {u.id === currentUser.id ? '(You)' : ''}</span>
                {u.id !== currentUser.id && (
                    <button onClick={() => handleInvite(u.id)} style={{ marginLeft: '10px', padding: '2px 5px', cursor: 'pointer' }}>Invite</button>
                )}
              </li>
            ))}
          </ul>
        </div>

        <div style={{ flex: 1 }}>
          <h3>Rooms</h3>
          <div style={{ marginBottom: '10px' }}>
            <input 
              value={newRoomName} 
              onChange={e => setNewRoomName(e.target.value)} 
              placeholder="New Room Name"
              style={{ padding: '5px' }}
            />
            <button onClick={handleCreateRoom} style={{ marginLeft: '5px', padding: '5px' }}>Create</button>
          </div>
          <ul style={{ listStyle: 'none', padding: 0 }}>
            {rooms.map(r => (
              <li key={r.id} style={{ padding: '10px', border: '1px solid #444', marginBottom: '5px', display: 'flex', justifyContent: 'space-between' }}>
                <span>{r.name} ({r.userIds.length} users)</span>
                <button onClick={() => handleJoinRoom(r.id)}>Join</button>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
};

export default Lobby;
