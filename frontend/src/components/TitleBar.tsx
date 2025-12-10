import React from 'react';
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime';

const TitleBar: React.FC = () => {
  return (
    <div className="title-bar">
      <div style={{ color: '#ecf0f1', fontSize: '14px', fontWeight: 'bold' }}>GoTeamWork</div>
      <div className="title-bar-controls">
        <button 
          onClick={WindowMinimise}
          style={{ background: 'transparent', border: 'none', color: '#bdc3c7', cursor: 'pointer' }}
        >
          _
        </button>
        <button 
          onClick={WindowToggleMaximise}
          style={{ background: 'transparent', border: 'none', color: '#bdc3c7', cursor: 'pointer' }}
        >
          □
        </button>
        <button 
          onClick={Quit}
          style={{ background: 'transparent', border: 'none', color: '#e74c3c', cursor: 'pointer' }}
        >
          ✕
        </button>
      </div>
    </div>
  );
};

export default TitleBar;
