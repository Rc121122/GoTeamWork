import React, { useState } from 'react';
import { hostCreateUser, hostSetUser } from '../api/wailsBridge';
import { httpCreateUser, setApiBaseUrl } from '../api/httpClient';

interface NewUserPageProps {
  onUserCreated: (user: { id: string; name: string }) => void;
  appMode: 'host' | 'client';
}

const NewUserPage: React.FC<NewUserPageProps> = ({ onUserCreated, appMode }) => {
  const [username, setUsername] = useState('');
  const [serverIp, setServerIp] = useState('localhost');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) return;

    if (appMode === 'client' && serverIp) {
      setApiBaseUrl(serverIp);
    }

    try {
      let user: { id: string; name: string };
      if (appMode === 'client') {
        // 1. Create user on Host
        user = await httpCreateUser({ name: username });
        // 2. Sync user to local Wails backend
        await hostSetUser(user.id, user.name);
      } else {
        user = await hostCreateUser(username);
      }
      onUserCreated(user);
    } catch (err: any) {
      console.error(err);
      if (err.status === 409) {
        setError('Username already exists');
      } else {
        setError('Failed to create user. Check server connection.');
      }
    }
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
      <h2>Create New User</h2>
      <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '10px', width: '300px' }}>
        {appMode === 'client' && (
          <input 
            type="text" 
            placeholder="Server IP (default: localhost)" 
            value={serverIp} 
            onChange={(e) => setServerIp(e.target.value)}
            style={{ padding: '10px', fontSize: '1rem' }}
          />
        )}
        <input 
          type="text" 
          placeholder="Enter Username" 
          value={username} 
          onChange={(e) => setUsername(e.target.value)}
          style={{ padding: '10px', fontSize: '1rem' }}
        />
        <button type="submit" style={{ padding: '10px', fontSize: '1rem' }}>Create</button>
      </form>
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  );
};

export default NewUserPage;
