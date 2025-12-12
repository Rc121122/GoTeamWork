import React, { useState } from 'react';
import { hostCreateUser, hostSetUser, hostSetServerURL } from '../api/wailsBridge';
import { httpCreateUser, setApiBaseUrl, parseServerUrl, setAuthToken } from '../api/httpClient';

interface NewUserPageProps {
  onUserCreated: (user: { id: string; name: string; token?: string }) => void;
  appMode: 'host' | 'client';
}

const NewUserPage: React.FC<NewUserPageProps> = ({ onUserCreated, appMode }) => {
  const [username, setUsername] = useState('');
  const [serverAddress, setServerAddress] = useState('');
  const [error, setError] = useState('');
  const [connecting, setConnecting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) {
      setError('Please enter a username');
      return;
    }

    setError('');
    setConnecting(true);

    try {
      let user: { id: string; name: string; token?: string };
      
      if (appMode === 'client') {
        // Parse and set server URL for both frontend HTTP client and backend network client
        const serverUrl = parseServerUrl(serverAddress);
        console.log('Connecting to server:', serverUrl);
        
        setApiBaseUrl(serverUrl); // Frontend HTTP client
        await hostSetServerURL(serverUrl); // Backend network client
        
        // Create user on remote Host server
        const resp = await httpCreateUser({ name: username });
        setAuthToken(resp.token);
        user = { ...resp.user, token: resp.token };
        // Sync user to local Wails backend
        await hostSetUser(user.id, user.name);
      } else {
        // Host mode - create user locally
        user = await hostCreateUser(username);
      }
      
      onUserCreated(user);
    } catch (err: any) {
      console.error('Connection error:', err);
      if (err.status === 409) {
        setError('Username already exists. Please choose another.');
      } else if (err.message?.includes('fetch') || err.message?.includes('network')) {
        setError('Cannot connect to server. Please check the address and try again.');
      } else {
        setError(`Connection failed: ${err.message || 'Unknown error'}`);
      }
    } finally {
      setConnecting(false);
    }
  };

  return (
    <div className="hero-shell">
      <div className="hero-card">
        <div className="hero-left">
          <p className="pill" style={{ display: 'inline-block' }}>Profile</p>
          <h1 style={{ marginBottom: '8px' }}>Create your profile</h1>
          <p style={{ color: 'var(--text-muted)', marginBottom: '16px' }}>
            {appMode === 'client' 
              ? 'Enter the server address and your username to connect.'
              : 'Choose a username to start hosting.'}
          </p>
          <div className="card-section">
            <form onSubmit={handleSubmit} className="username-input-group" style={{ margin: 0 }}>
              {appMode === 'client' && (
                <>
                  <input
                    type="text"
                    placeholder="Server Address (leave empty for localhost)"
                    value={serverAddress}
                    onChange={(e) => setServerAddress(e.target.value)}
                    className="text-input"
                    disabled={connecting}
                  />
                  <p style={{ fontSize: '12px', color: 'var(--text-muted)', margin: '4px 0 12px 0' }}>
                    Examples: <code>192.168.1.100</code> (LAN) or <code>https://xxx.trycloudflare.com</code> (tunnel)
                  </p>
                </>
              )}
              <input
                type="text"
                placeholder="Enter Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="text-input"
                disabled={connecting}
              />
              <button type="submit" className="primary-btn" disabled={connecting}>
                {connecting ? 'Connecting...' : 'Connect'}
              </button>
            </form>
            {error && <p className="error-message" style={{ display: 'block', marginTop: '12px' }}>{error}</p>}
          </div>
        </div>
        <div className="card-section">
          <h3 style={{ marginTop: 0 }}>Connection Guide</h3>
          <ul style={{ margin: '12px 0 0 16px', color: 'var(--text-muted)', lineHeight: 1.8 }}>
            <li><strong>Same machine:</strong> Leave address empty or use <code>localhost</code></li>
            <li><strong>LAN:</strong> Use host's IP address (e.g., <code>192.168.1.100</code>)</li>
            <li><strong>Internet (Cloudflare):</strong> Use the full tunnel URL</li>
            <li>Usernames must be unique across the server</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default NewUserPage;
