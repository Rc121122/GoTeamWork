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
    <div className="modal-backdrop">
      <div className="modal-card">
        <div className="modal-head">
          <h3 style={{ margin: 0 }}>{title}</h3>
          <button onClick={onClose} className="modal-close">✕</button>
        </div>
        <div>{children}</div>
      </div>
    </div>
  );
};

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  isHUDEnabled: boolean;
  onToggleHUD: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose, isHUDEnabled, onToggleHUD }) => (
  <Modal isOpen={isOpen} onClose={onClose} title="Settings">
    <div style={{ marginBottom: '20px' }}>
      <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
        <input 
          type="checkbox" 
          checked={isHUDEnabled} 
          onChange={onToggleHUD}
          style={{ marginRight: '10px', transform: 'scale(1.2)' }}
        />
        <span>Enable Clipboard Sharing HUD</span>
      </label>
      <p style={{ fontSize: '0.8rem', color: '#bdc3c7', marginLeft: '25px', marginTop: '5px' }}>
        When enabled, a small popup will appear when you copy text/files, allowing you to share them.
      </p>
    </div>
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
