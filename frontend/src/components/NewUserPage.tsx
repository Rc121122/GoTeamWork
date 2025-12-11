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
    <div className="hero-shell">
      <div className="hero-card">
        <div className="hero-left">
          <p className="pill" style={{ display: 'inline-block' }}>Profile</p>
          <h1 style={{ marginBottom: '8px' }}>Create your profile</h1>
          <p style={{ color: 'var(--text-muted)', marginBottom: '16px' }}>
            Choose a username and connect to your host server.
          </p>
          <div className="card-section">
            <form onSubmit={handleSubmit} className="username-input-group" style={{ margin: 0 }}>
              {appMode === 'client' && (
                <input
                  type="text"
                  placeholder="Server IP (default: localhost)"
                  value={serverIp}
                  onChange={(e) => setServerIp(e.target.value)}
                  className="text-input"
                />
              )}
              <input
                type="text"
                placeholder="Enter Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="text-input"
              />
              <button type="submit" className="primary-btn">Create</button>
            </form>
            {error && <p className="error-message" style={{ display: 'block' }}>{error}</p>}
          </div>
        </div>
        <div className="card-section">
          <h3 style={{ marginTop: 0 }}>Tips</h3>
          <ul style={{ margin: '12px 0 0 16px', color: 'var(--text-muted)', lineHeight: 1.6 }}>
            <li>Use your host machine IP for client connections.</li>
            <li>Usernames must be unique; conflicts return 409.</li>
            <li>After creation, you will enter the lobby to join or create rooms.</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default NewUserPage;
