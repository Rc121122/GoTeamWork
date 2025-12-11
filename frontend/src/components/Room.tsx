import React, { useState, useEffect, useRef } from 'react';
import { hostSendChatMessage, hostFetchChatHistory, hostLeaveRoom, hostFetchOperations } from '../api/wailsBridge';
import { httpSendChatMessage, httpFetchChatHistory, httpLeaveRoom, httpFetchOperations } from '../api/httpClient';
import { ChatMessage, Room, Operation, CopiedItem } from '../api/types';
import { addSSEListener, removeSSEListener } from '../sse';
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

interface RoomProps {
  currentUser: { id: string; name: string };
  currentRoom: Room;
  onLeave: () => void;
  appMode: 'host' | 'client';
}

const RoomView: React.FC<RoomProps> = ({ currentUser, currentRoom, onLeave, appMode }) => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [operations, setOperations] = useState<Operation[]>([]);
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

  const refreshOperations = async () => {
    try {
      let ops: Operation[];
      const roomIdToFetch = currentRoom.id; 
      
      if (appMode === 'client') {
        ops = await httpFetchOperations(roomIdToFetch);
      } else {
        ops = await hostFetchOperations(roomIdToFetch);
      }
      // Filter for clipboard items only
      const clipboardOps = ops.filter(op => op.item.type === 'clipboard');
      setOperations(clipboardOps);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    refreshChat();
    refreshOperations();

    const onChatMsg = (msg: ChatMessage) => {
        if (msg.roomId === currentRoom.id) {
            setMessages(prev => [...prev, msg]);
        }
    };

    const onClipboard = (payload: CopiedItem | Operation) => {
        console.log("Received clipboard payload via SSE:", payload);
        
        let newOp: Operation;

        // Check if payload is an Operation
        if ('opType' in payload && 'item' in payload) {
             newOp = payload as Operation;
        } else {
             // Fallback for direct item broadcast
             const item = payload as CopiedItem;
             newOp = {
                id: `temp_${Date.now()}`,
                parentId: "",
                opType: "add",
                itemId: `temp_item_${Date.now()}`,
                item: {
                    id: `temp_item_${Date.now()}`,
                    type: "clipboard",
                    data: item
                },
                timestamp: Date.now() / 1000
            };
        }
        setOperations(prev => {
            if (prev.some(o => o.id === newOp.id)) return prev;
            return [...prev, newOp];
        });
    };

    const onClipboardUpdated = (op: Operation) => {
        console.log("Received clipboard update via SSE:", op);
        if (op.item && op.item.data) {
             const data = op.item.data as CopiedItem;
             console.log("Updated item text:", data.text);
        } else {
             console.warn("Received update with no item data:", op);
        }
        setOperations(prev => {
            const exists = prev.some(o => o.id === op.id || o.itemId === op.itemId);
            if (exists) {
                return prev.map(o => {
                    if (o.id === op.id || o.itemId === op.itemId) {
                        return op;
                    }
                    return o;
                });
            } else {
                return [...prev, op];
            }
        });
    };

    addSSEListener('chat_message', onChatMsg);
    addSSEListener('clipboard_copied', onClipboard);
    addSSEListener('clipboard_updated', onClipboardUpdated);

    return () => {
        removeSSEListener('chat_message', onChatMsg);
        removeSSEListener('clipboard_copied', onClipboard);
        removeSSEListener('clipboard_updated', onClipboardUpdated);
    };
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
      // refreshChat(); // No longer needed as we receive our own message via SSE
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

  const renderClipboardItem = (op: Operation) => {
      const item = op.item.data as CopiedItem;
      if (!item) {
          console.log("Clipboard item data is null for operation:", op.id);
          return null;
      }

      console.log("Rendering clipboard item:", item.type, item);

      return (
          <div key={op.id} style={{ background: 'white', color: 'black', padding: '10px', borderRadius: '5px', width: '100%', marginBottom: '10px' }}>
              <div style={{ fontSize: '0.8rem', color: '#666', marginBottom: '5px', display: 'flex', alignItems: 'center' }}>
                  <span style={{ marginRight: '5px' }}>
                      {item.type === 'text' ? '📄' : item.type === 'image' ? '🖼️' : item.type === 'file' ? '📁' : '❓'}
                  </span>
                  <span style={{ fontWeight: 'bold', marginRight: '5px' }}>
                      {op.userName || 'Unknown'}
                  </span>
                  {new Date(op.timestamp * 1000).toLocaleTimeString()}
              </div>
              {item.type === 'text' && (
                  <div style={{ whiteSpace: 'pre-wrap', maxHeight: '100px', overflowY: 'auto' }}>
                      {item.text}
                  </div>
              )}
              {item.type === 'image' && item.image && (
                  <img src={`data:image/png;base64,${item.image}`} alt="Clipboard" style={{ maxWidth: '100%', maxHeight: '150px' }} />
              )}
              {item.type === 'file' && item.files && (
                  <div>
                      <strong>Files:</strong>
                      <ul style={{ margin: 0, paddingLeft: '20px' }}>
                          {item.files.map((file, idx) => (
                              <li key={idx} style={{ fontSize: '0.9rem' }}>{file}</li>
                          ))}
                      </ul>
                      <div style={{ marginTop: '10px' }}>
                          {item.text && item.text.includes('(ready)') ? (
                              <a 
                                  href="#"
                                  onClick={(e) => {
                                      e.preventDefault();
                                      BrowserOpenURL(`http://localhost:8080/api/download/${op.id}`);
                                  }}
                                  style={{
                                      display: 'inline-block',
                                      padding: '5px 10px',
                                      background: '#3498db',
                                      color: 'white',
                                      textDecoration: 'none',
                                      borderRadius: '3px',
                                      fontSize: '0.9rem',
                                      cursor: 'pointer'
                                  }}
                              >
                                  Download Zip
                              </a>
                          ) : (
                              <span style={{ color: '#888', fontSize: '0.9rem' }}>
                                  Compressing files...
                              </span>
                          )}
                      </div>
                  </div>
              )}
          </div>
      );
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
              <strong>{msg.userId === currentUser.id ? 'Me' : msg.userName}:</strong> {msg.message}
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
        <div style={{ marginTop: '10px', padding: '5px', borderTop: '1px solid #444', fontSize: '0.9rem', color: '#aaa' }}>
            Logged in as: <strong>{currentUser.name}</strong>
        </div>
      </div>

      {/* Right Column: Shared Clipboard */}
      <div style={{ flex: 1, padding: '10px', background: '#2c3e50', overflowY: 'auto' }}>
        <h3>Shared Clipboard</h3>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
          {operations.length === 0 ? (
             <div style={{ background: 'rgba(255,255,255,0.1)', padding: '10px', borderRadius: '5px' }}>
                <p>No shared items yet.</p>
             </div>
          ) : (
              operations.slice().reverse().map(renderClipboardItem)
          )}
        </div>
      </div>
    </div>
  );
};

export default RoomView;
