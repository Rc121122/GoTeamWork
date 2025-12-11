import React from 'react';
import gopherIdle from '../assets/gopher/gopher-idle.png';
import gopherCarry from '../assets/gopher/gopher-carry.png';
import { ShareSystemClipboard } from '../../wailsjs/go/main/App';

interface HUDProps {
  onClose: () => void;
}

const HUD: React.FC<HUDProps> = ({ onClose }) => {
  const [state, setState] = React.useState<'idle' | 'carry'>('idle');

  // Auto-hide after 5 seconds
  React.useEffect(() => {
    const timer = setTimeout(() => {
      onClose();
    }, 5000);
    return () => clearTimeout(timer);
  }, [onClose]);

  const handleClick = async () => {
    if (state === 'idle') {
      try {
        console.log("Sharing system clipboard...");
        setState('carry');
        await ShareSystemClipboard();
        console.log("ShareSystemClipboard done");
        onClose();
      } catch (err) {
        console.error("Error sharing clipboard:", err);
        onClose();
      }
    } else {
        onClose();
    }
  };

  return (
    <div style={{ 
        position: 'fixed',
        inset: 0,
        width: '100%', 
        height: '100%', 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        pointerEvents: 'auto',
        background: '#1e293b', // Solid dark background - no transparency
        zIndex: 9999,
        WebkitAppRegion: 'drag', // Allow dragging
    }}>
      <div style={{
        width: '140px',
        height: '140px',
        borderRadius: '16px',
        background: 'linear-gradient(135deg, #334155 0%, #1e293b 100%)',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        boxShadow: '0 4px 20px rgba(0,0,0,0.3)',
      }}>
        <img 
          src={state === 'idle' ? gopherIdle : gopherCarry} 
          alt="Gopher" 
          style={{ width: '100px', cursor: 'pointer', WebkitAppRegion: 'no-drag' } as React.CSSProperties} 
          onClick={handleClick}
        />
      </div>
    </div>
  );
};

export default HUD;
