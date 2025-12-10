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
        await ShareSystemClipboard();
        setState('carry');
        // After sharing, we might want to close or wait.
        // For now, let's close after a short delay to indicate success
        setTimeout(() => {
            onClose();
        }, 1000);
      } catch (err) {
        console.error(err);
      }
    } else {
        onClose();
    }
  };

  return (
    <div style={{ 
        width: '100%', 
        height: '100%', 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        background: 'transparent', // Ensure transparent background
        // @ts-ignore
        WebkitAppRegion: 'drag' // Allow dragging
    }}>
      <img 
        src={state === 'idle' ? gopherIdle : gopherCarry} 
        alt="Gopher" 
        style={{ width: '100px', cursor: 'pointer', WebkitAppRegion: 'no-drag' } as React.CSSProperties} 
        onClick={handleClick}
      />
    </div>
  );
};

export default HUD;
