import React from 'react';

interface LandingPageProps {
  onStart: () => void;
}

const LandingPage: React.FC<LandingPageProps> = ({ onStart }) => {
  return (
    <div className="hero-shell">
      <div className="hero-card">
        <div className="hero-left">
          <p className="pill" style={{ display: 'inline-block' }}>Welcome</p>
          <h1>GoTeamWork</h1>
          <p>Share clips, chat, and collaborate across rooms with a lightweight, real-time workflow.</p>
          <div className="pill-row">
            <span className="pill">Low-latency SSE</span>
            <span className="pill">Clipboard sharing</span>
            <span className="pill">Invite & approvals</span>
          </div>
          <div style={{ display: 'flex', gap: '12px', marginTop: '10px' }}>
            <button className="primary-btn" onClick={onStart}>Start</button>
          </div>
        </div>
        <div className="card-section" style={{ alignSelf: 'stretch' }}>
          <h3 style={{ marginTop: 0 }}>How it works</h3>
          <ul style={{ margin: '12px 0 0 16px', color: 'var(--text-muted)', lineHeight: 1.6 }}>
            <li>Create your user profile.</li>
            <li>Request to join a room or accept an invite.</li>
            <li>Share files via clipboard zip uploads and chat in real time.</li>
          </ul>
        </div>
      </div>
    </div>
  );
};

export default LandingPage;
