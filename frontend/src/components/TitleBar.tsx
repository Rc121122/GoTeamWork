import React from 'react';
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime';

const TitleBar: React.FC = () => {
  const isMac = typeof navigator !== 'undefined' && navigator.userAgent.toLowerCase().includes('mac');

  return (
    <div className={`title-bar ${isMac ? 'title-bar-mac' : ''}`}>
      {isMac ? (
        <div className="mac-header" style={{ WebkitAppRegion: 'no-drag', '--wails-draggable': 'no-drag' } as React.CSSProperties}>
          <span className="red" onClick={Quit} />
          <span className="yellow" onClick={WindowMinimise} />
          <span className="green" onClick={WindowToggleMaximise} />
          <div className="mac-title">GoTeamWork</div>
        </div>
      ) : (
        <>
          <div className="title-text">GoTeamWork</div>
          <div className="title-bar-controls">
            <button 
              onClick={WindowMinimise}
              className="title-btn"
            >
              _
            </button>
            <button 
              onClick={WindowToggleMaximise}
              className="title-btn"
            >
              □
            </button>
            <button 
              onClick={Quit}
              className="title-btn close"
            >
              ✕
            </button>
          </div>
        </>
      )}
    </div>
  );
};

export default TitleBar;
