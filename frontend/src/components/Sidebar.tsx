import React from 'react';

interface SidebarProps {
  onReboot: () => void;
  onSettings: () => void;
  onAbout: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({ onReboot, onSettings, onAbout }) => {
  return (
    <nav className="sidebar">
      <div className="logo">GTW</div>
      <button onClick={onReboot} title="Reboot">↻</button>
      <button onClick={onSettings} title="Settings">⚙</button>
      <button onClick={onAbout} title="About">ℹ</button>
    </nav>
  );
};

export default Sidebar;
