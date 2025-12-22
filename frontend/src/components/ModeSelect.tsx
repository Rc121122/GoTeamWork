import React from 'react';

interface ModeSelectProps {
  onSelect: (mode: 'host' | 'client') => void;
}

const ModeSelect: React.FC<ModeSelectProps> = ({ onSelect }) => {
  return (
    <div className="hero-shell">
      <div className="hero-card" style={{ gap: '16px', maxWidth: '720px' }}>
        <div className="hero-left">
          <p className="pill" style={{ display: 'inline-block' }}>Choose a role</p>
          <h1>How do you want to run?</h1>
          <p>Pick host to run the server and share from your machine, or client to join an existing host.</p>
          <div style={{ display: 'flex', gap: '12px', marginTop: '10px', flexWrap: 'wrap' }}>
            <button className="primary-btn" onClick={() => onSelect('client')}>Join as Client</button>
            <button className="secondary-btn" onClick={() => onSelect('host')}>Run as Host</button>
          </div>
        </div>
        <div className="card-section" style={{ alignSelf: 'stretch' }}>
          <h3 style={{ marginTop: 0 }}>What each mode does</h3>
          <ul style={{ margin: '12px 0 0 16px', color: 'var(--text-muted)', lineHeight: 1.6 }}>
            <li><strong>Host</strong>: Starts the server, serves REST/SSE, accepts join requests.</li>
            <li><strong>Client</strong>: Discovers/joins hosts, shares clipboard via rooms.</li>
            <li>You can restart and pick a different mode anytime.</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default ModeSelect;
