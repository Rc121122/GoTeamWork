import React from 'react';
import gopherOriginal from '../assets/gopher/gopher_original.png';
import gopherText from '../assets/gopher/gopher_text.png';
import gopherImage from '../assets/gopher/gopher_image.png';
import gopherFolder from '../assets/gopher/gopher_folder.png';
import gopherCarry from '../assets/gopher/gopher-carry.png';
import { ShareSystemClipboard } from '../../wailsjs/go/main/App';

interface HUDProps {
  onClose: () => void;
  contentType?: string;
}

const HUD: React.FC<HUDProps> = ({ onClose, contentType = 'text' }) => {
  const [state, setState] = React.useState<'idle' | 'carry'>('idle');
  const [showNotification, setShowNotification] = React.useState(false);

  // Set body background to transparent when HUD mounts
  React.useEffect(() => {
    document.body.style.background = 'transparent';
    return () => {
      document.body.style.background = ''; // Reset on unmount
    };
  }, []);

  // Auto-hide after 5 seconds
  React.useEffect(() => {
    const timer = setTimeout(() => {
      onClose();
    }, 5000);
    return () => clearTimeout(timer);
  }, [onClose]);

  const getIconForContentType = (type: string, state: 'idle' | 'carry') => {
    if (state === 'carry') return gopherCarry;
    
    switch (type) {
      case 'text': return gopherText;
      case 'image': return gopherImage;
      case 'file': return gopherFolder;
      default: return gopherOriginal;
    }
  };

  const handleClick = async () => {
    if (state === 'idle') {
      try {
        console.log("Sharing system clipboard...");
        setState('carry');
        await ShareSystemClipboard();
        console.log("ShareSystemClipboard done");
        
        // Show notification
        setShowNotification(true);
        setTimeout(() => setShowNotification(false), 2000);
        
        // Close after a brief delay to show the notification
        setTimeout(() => onClose(), 1000);
      } catch (err) {
        console.error("Error sharing clipboard:", err);
        onClose();
      }
    } else {
        onClose();
    }
  };

  return (
    <div 
      onMouseDown={onClose}
      style={{ 
        position: 'fixed',
        inset: 0,
        width: '100%', 
        height: '100%', 
        display: 'flex', 
        justifyContent: 'center', 
        alignItems: 'center', 
        pointerEvents: 'auto',
        background: 'transparent',
        zIndex: 9999,
        // WebkitAppRegion: 'drag', // Allow dragging
      }}>
      <img 
        src={getIconForContentType(contentType, state)} 
        alt="Gopher" 
        style={{ width: '40px', cursor: 'pointer', marginLeft: '100px', WebkitAppRegion: 'no-drag', opacity: 0.9 } as React.CSSProperties} 
        onMouseDown={(e) => e.stopPropagation()}
        onClick={handleClick}
      />
      
      {/* Item shared notification */}
      {showNotification && (
        <div style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          background: 'rgba(0, 0, 0, 0.8)',
          color: 'white',
          padding: '8px 16px',
          borderRadius: '4px',
          fontSize: '14px',
          fontWeight: 'bold',
          zIndex: 10000,
          animation: 'fadeIn 0.3s ease-in-out'
        }}>
          Item shared!
        </div>
      )}
      
      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; transform: translate(-50%, -60%); }
          to { opacity: 1; transform: translate(-50%, -50%); }
        }
      `}</style>
    </div>
  );
};

export default HUD;
