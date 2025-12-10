import React from 'react';

interface LandingPageProps {
  onStart: () => void;
}

const LandingPage: React.FC<LandingPageProps> = ({ onStart }) => {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
      <h1>Welcome to GoTeamWork</h1>
      <p>Collaborate seamlessly with your team.</p>
      <button onClick={onStart} style={{ padding: '10px 20px', fontSize: '1.2rem', marginTop: '20px' }}>
        Start
      </button>
    </div>
  );
};

export default LandingPage;
