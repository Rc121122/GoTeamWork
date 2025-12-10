import React, { useEffect, useState } from 'react';
import { hostListUsers, hostListRooms } from '../api/wailsBridge';
import { User, Room } from '../api/types';

const HostDashboard: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [rooms, setRooms] = useState<Room[]>([]);

  const refreshData = async () => {
    try {
      const u = await hostListUsers();
      const r = await hostListRooms();
      setUsers(u);
      setRooms(r);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    refreshData();
    const interval = setInterval(refreshData, 2000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div style={{ display: 'flex', height: '100%' }}>
      {/* Left Column: Users */}
      <div style={{ flex: 1, borderRight: '1px solid #444', padding: '20px', overflowY: 'auto' }}>
        <h2>All Users</h2>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {users.map(u => (
            <li key={u.id} style={{ padding: '10px', borderBottom: '1px solid #555' }}>
              <strong>{u.name}</strong> ({u.isOnline ? 'Online' : 'Offline'})
              <br/>
              <small>ID: {u.id}</small>
            </li>
          ))}
        </ul>
      </div>

      {/* Right Column: Rooms */}
      <div style={{ flex: 1, padding: '20px', overflowY: 'auto' }}>
        <h2>All Rooms</h2>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {rooms.map(r => (
            <li key={r.id} style={{ padding: '10px', border: '1px solid #555', marginBottom: '10px', borderRadius: '5px' }}>
              <h3>{r.name}</h3>
              <p>Users: {r.userIds.length}</p>
              <small>ID: {r.id}</small>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};

export default HostDashboard;
