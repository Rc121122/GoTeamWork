import React from 'react';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

const Modal: React.FC<ModalProps> = ({ isOpen, onClose, title, children }) => {
  if (!isOpen) return null;

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
      background: 'rgba(0,0,0,0.5)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000
    }}>
      <div style={{
        background: '#34495e', padding: '20px', borderRadius: '10px', width: '400px', color: 'white',
        boxShadow: '0 4px 6px rgba(0,0,0,0.1)'
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '15px' }}>
          <h3>{title}</h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: 'white', fontSize: '1.2rem', cursor: 'pointer' }}>✕</button>
        </div>
        <div>{children}</div>
      </div>
    </div>
  );
};

export const SettingsModal: React.FC<{ isOpen: boolean; onClose: () => void }> = (props) => (
  <Modal {...props} title="Settings">
    <p>App settings will go here.</p>
    <p>Version: 1.0.0</p>
  </Modal>
);

export const AboutModal: React.FC<{ isOpen: boolean; onClose: () => void }> = (props) => (
  <Modal {...props} title="About">
    <p>GoTeamWork</p>
    <p>Collaborate seamlessly.</p>
    <a href="https://github.com/Rc121122/GoTeamWork" target="_blank" rel="noreferrer" style={{ color: '#3498db' }}>GitHub Repo</a>
  </Modal>
);
