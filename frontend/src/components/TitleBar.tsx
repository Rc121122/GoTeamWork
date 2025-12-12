import React from 'react';
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime';

const TitleBar: React.FC = () => {
  return (
    <div className="title-bar">
      <div style={{ color: '#ecf0f1', fontSize: '14px', fontWeight: 'bold' }}>GoTeamWork</div>
      <div className="title-bar-controls">
        <button 
          onClick={WindowMinimise}
          style={{ background: 'transparent', border: 'none', color: '#bdc3c7', cursor: 'pointer', marginBottom: '2px' }}
        >
          _
        </button>
        <button 
          onClick={WindowToggleMaximise}
          style={{ background: 'transparent', border: 'none', color: '#bdc3c7', cursor: 'pointer', marginTop: '2px'}}
        >
          □
        </button>
        <button 
          onClick={Quit}
          style={{ background: 'transparent', border: 'none', color: '#e74c3c', cursor: 'pointer', fontSize: '1.2em' }}
        >
          ✕
        </button>
      </div>
    </div>
  );
};

export default TitleBar;
