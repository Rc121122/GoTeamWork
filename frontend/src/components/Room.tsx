import React, { useState, useEffect, useRef } from 'react';
import { hostSendChatMessage, hostFetchChatHistory, hostLeaveRoom } from '../api/wailsBridge';
import { httpSendChatMessage, httpFetchChatHistory, httpLeaveRoom } from '../api/httpClient';
import { ChatMessage, Room } from '../api/types';

interface RoomProps {
  currentUser: { id: string; name: string };
  currentRoom: Room;
  onLeave: () => void;
  appMode: 'host' | 'client';
}

const RoomView: React.FC<RoomProps> = ({ currentUser, currentRoom, onLeave, appMode }) => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const chatEndRef = useRef<HTMLDivElement>(null);

  const refreshChat = async () => {
    try {
      let history: ChatMessage[];
      if (appMode === 'client') {
        history = await httpFetchChatHistory(currentRoom.id);
      } else {
        history = await hostFetchChatHistory(currentRoom.id);
      }
      setMessages(history);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    refreshChat();
    const interval = setInterval(refreshChat, 1000); // Poll chat
    return () => clearInterval(interval);
  }, [currentRoom.id]);

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newMessage.trim()) return;
    try {
      if (appMode === 'client') {
        await httpSendChatMessage({ roomId: currentRoom.id, userId: currentUser.id, message: newMessage });
      } else {
        await hostSendChatMessage(currentRoom.id, currentUser.id, newMessage);
      }
      setNewMessage('');
      refreshChat();
    } catch (err) {
      console.error(err);
    }
  };

  const handleLeave = async () => {
      try {
          if (appMode === 'client') {
            await httpLeaveRoom({ userId: currentUser.id });
          } else {
            await hostLeaveRoom(currentUser.id);
          }
          onLeave();
      } catch (err) {
          console.error("Failed to leave room", err);
      }
  };

  return (
    <div style={{ display: 'flex', height: '100%' }}>
      {/* Left Column: Chat */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', borderRight: '1px solid #444', padding: '10px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
            <h3>Chat: {currentRoom.name}</h3>
            <button onClick={handleLeave} style={{background: '#e74c3c', color: 'white', border: 'none', padding: '5px 10px', cursor: 'pointer'}}>Leave Room</button>
        </div>
        
        <div style={{ flex: 1, overflowY: 'auto', marginBottom: '10px', background: 'rgba(0,0,0,0.2)', padding: '10px', borderRadius: '5px' }}>
          {messages.map(msg => (
            <div key={msg.id} style={{ marginBottom: '5px' }}>
              <strong>{msg.userName}:</strong> {msg.message}
            </div>
          ))}
          <div ref={chatEndRef} />
        </div>
        <form onSubmit={handleSend} style={{ display: 'flex' }}>
          <input 
            value={newMessage} 
            onChange={e => setNewMessage(e.target.value)} 
            style={{ flex: 1, padding: '10px' }}
            placeholder="Type a message..."
          />
          <button type="submit" style={{ padding: '10px' }}>Send</button>
        </form>
      </div>

      {/* Right Column: Shared Clipboard */}
      <div style={{ flex: 1, padding: '10px', background: '#2c3e50' }}>
        <h3>Shared Clipboard</h3>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
          {/* Placeholder for clipboard cards */}
          <div style={{ background: 'white', color: 'black', padding: '10px', borderRadius: '5px', width: '100%' }}>
            <p>Clipboard history will appear here...</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default RoomView;
