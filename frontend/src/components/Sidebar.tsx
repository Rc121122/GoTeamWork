import React from 'react';

interface SidebarProps {
  onReboot: () => void;
  onSettings: () => void;
  onAbout: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ onReboot, onSettings, onAbout }) => {
  return (
    <nav style={{
      width: '60px', 
      background: '#2c3e50', 
      display: 'flex', 
      flexDirection: 'column', 
      alignItems: 'center', 
      padding: '1rem 0',
      height: '100vh'
    }}>
      <div style={{ marginBottom: 'auto' }}>
        {/* Logo or Icon could go here */}
        <div style={{ color: 'white', fontWeight: 'bold', fontSize: '1.5rem' }}>GTW</div>
      </div>
      
      <button onClick={onReboot} title="Reboot" style={btnStyle}>↻</button>
      <button onClick={onSettings} title="Settings" style={btnStyle}>⚙</button>
      <button onClick={onAbout} title="About" style={btnStyle}>ℹ</button>
    </nav>
  );
};

const btnStyle: React.CSSProperties = {
  background: 'transparent',
  border: 'none',
  color: 'white',
  fontSize: '1.5rem',
  cursor: 'pointer',
  margin: '1rem 0',
  padding: '0.5rem',
  width: '100%',
  textAlign: 'center',
};

export default Sidebar;
