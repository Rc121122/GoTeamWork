import React, { useState, useEffect, useRef } from 'react';
import { hostSendChatMessage, hostFetchChatHistory, hostLeaveRoom, hostFetchOperations, hostInviteUser } from '../api/wailsBridge';
import { httpSendChatMessage, httpFetchChatHistory, httpLeaveRoom, httpFetchOperations, getApiBaseUrl, httpFetchUsers, httpInviteUser } from '../api/httpClient';
import { ChatMessage, Room, Operation, CopiedItem, User } from '../api/types';
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
  const [inviteOpen, setInviteOpen] = useState(false);
  const [inviteUsers, setInviteUsers] = useState<User[]>([]);
  const [inviteLoading, setInviteLoading] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [invitedIds, setInvitedIds] = useState<Set<string>>(new Set());
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
            setMessages(prev => {
                // Check if this is our own message that we already added locally
                const existingIndex = prev.findIndex(m => 
                    m.userId === msg.userId && 
                    m.message === msg.message && 
                    Math.abs(m.timestamp - msg.timestamp) < 10 // within 10 seconds
                );
                if (existingIndex !== -1) {
                    // Replace the local message with the server message
                    const newMessages = [...prev];
                    newMessages[existingIndex] = msg;
                    return newMessages;
                } else {
                    return [...prev, msg];
                }
            });
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
    const messageToSend = newMessage.trim();
    try {
      if (appMode === 'client') {
        await httpSendChatMessage({ roomId: currentRoom.id, userId: currentUser.id, message: messageToSend });
      } else {
        await hostSendChatMessage(currentRoom.id, currentUser.id, messageToSend);
      }
      // Immediately add the message to local state
      const sentMessage: ChatMessage = {
        id: `sent_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
        roomId: currentRoom.id,
        userId: currentUser.id,
        userName: currentUser.name,
        message: messageToSend,
        timestamp: Date.now() / 1000,
      };
      setMessages(prev => [...prev, sentMessage]);
      setNewMessage('');
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

  const formatBytes = (bytes?: number) => {
    if (!bytes) return '';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
      value /= 1024;
      unitIndex++;
    }
    return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
  };

  const getFileThumb = (name?: string, fallback?: string) => {
    if (!name && !fallback) return 'FILE';
    const source = fallback || name || '';
    const ext = source.includes('.') ? source.split('.').pop() : source;
    const label = (ext || 'FILE').toUpperCase();
    return label.slice(0, 4);
  };

  const renderClipboardItem = (op: Operation) => {
      const item = op.item.data as CopiedItem;
      if (!item) {
          console.log("Clipboard item data is null for operation:", op.id);
          return null;
      }

      const handleDragStart = (event: React.DragEvent<HTMLDivElement>) => {
        event.dataTransfer.effectAllowed = 'copy';
        if (item.type === 'text' && item.text) {
          event.dataTransfer.setData('text/plain', item.text);
        } else if (item.type === 'image' && item.text) {
          // Fallback: share descriptive text for image
          event.dataTransfer.setData('text/plain', '[Image] ' + item.text);
        } else if (item.type === 'file') {
          // Provide a hint to the drop target; real file transfer requires native integration
          event.dataTransfer.setData('text/plain', 'Files ready to share');
        }
      };

      console.log("Rendering clipboard item:", item.type, item);

      const getTypeIcon = (type: string) => {
          switch (type) {
              case 'text': return '📄';
              case 'image': return '🖼️';
              case 'file': return '📁';
              default: return '❓';
          }
      };

      const getTypeColor = (type: string) => {
          switch (type) {
              case 'text': return '#3498db';
              case 'image': return '#e74c3c';
              case 'file': return '#27ae60';
              default: return '#95a5a6';
          }
      };

      const isSingleFile = item.type === 'file' && (item as any).isSingleFile;
      const singleFileName = (item as any)?.singleFileName as string | undefined;
      const singleFileSize = (item as any)?.singleFileSize as number | undefined;
      const singleFileThumb = getFileThumb(singleFileName, (item as any)?.singleFileThumb as string | undefined);

        const hasErrorText = item.text && (item.text.includes('too large') || item.text.includes('exceeds limit'));
        const readyByText = item.text && item.text.includes('(ready)');
        const readySingleFile = isSingleFile && typeof singleFileSize === 'number' && singleFileSize > 0;
        const downloadReady = Boolean(readyByText || readySingleFile);
        const downloadOpId = op.id || op.itemId;

        return (
          <div key={op.id} className="clipboard-card" draggable onDragStart={handleDragStart} style={{
              background: 'rgba(255, 255, 255, 0.05)',
              border: '1px solid rgba(255, 255, 255, 0.08)',
              borderRadius: '12px',
              padding: '16px',
              marginBottom: '12px',
              boxShadow: '0 2px 8px rgba(0,0,0,0.2)',
              transition: 'all 0.2s ease',
              cursor: 'default',
              position: 'relative',
              overflow: 'hidden'
          }}>
              {/* Header */}
              <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  marginBottom: '12px',
                  fontSize: '0.85rem',
                  color: '#94a3b8'
              }}>
                  <div style={{
                      width: '32px',
                      height: '32px',
                      borderRadius: '50%',
                      background: getTypeColor(item.type),
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '1.2rem',
                      marginRight: '10px',
                      boxShadow: '0 2px 4px rgba(0,0,0,0.3)'
                  }}>
                      {getTypeIcon(item.type)}
                  </div>
                  <div style={{ flex: 1 }}>
                      <div style={{ fontWeight: '600', color: '#e2e8f0', marginBottom: '2px' }}>
                          {op.userName || 'Unknown User'}
                      </div>
                      <div style={{ fontSize: '0.75rem' }}>
                          {new Date(op.timestamp * 1000).toLocaleString()}
                      </div>
                  </div>
              </div>

              {/* Content */}
              <div style={{ marginBottom: '12px' }}>
                  {item.type === 'text' && (
                      <div style={{
                          background: 'rgba(255, 255, 255, 0.03)',
                          borderRadius: '8px',
                          padding: '12px',
                          fontFamily: 'monospace',
                          fontSize: '0.9rem',
                          color: '#e2e8f0',
                          whiteSpace: 'pre-wrap',
                          maxHeight: '120px',
                          overflowY: 'auto',
                          border: '1px solid rgba(255, 255, 255, 0.08)'
                      }}>
                          {item.text}
                      </div>
                  )}
                  {item.type === 'image' && item.image && (
                      <div style={{
                          borderRadius: '8px',
                          overflow: 'hidden',
                          border: '1px solid rgba(255, 255, 255, 0.08)',
                          background: 'rgba(255, 255, 255, 0.02)'
                      }}>
                          <img
                              src={`data:image/png;base64,${item.image}`}
                              alt="Shared Image"
                              style={{
                                  maxWidth: '100%',
                                  maxHeight: '200px',
                                  display: 'block',
                                  borderRadius: '8px'
                              }}
                          />
                      </div>
                  )}
                    {item.type === 'file' && item.files && (
                      <div>
                        {isSingleFile ? (
                        <div className="file-thumbnail-row">
                          <div className="file-thumb-circle">
                          {singleFileThumb}
                          </div>
                          <div>
                          <div className="file-thumb-name">{singleFileName}</div>
                          {singleFileSize && (
                            <div className="file-thumb-size">{formatBytes(singleFileSize)}</div>
                          )}
                          </div>
                        </div>
                        ) : (
                        <div style={{
                          background: 'rgba(255, 255, 255, 0.03)',
                          borderRadius: '8px',
                          padding: '12px',
                          marginBottom: '12px',
                          border: '1px solid rgba(255, 255, 255, 0.08)'
                        }}>
                          <div style={{ fontWeight: '600', color: '#e2e8f0', marginBottom: '8px' }}>
                            📁 Files ({item.files.length})
                          </div>
                          <div style={{
                            maxHeight: '100px',
                            overflowY: 'auto',
                            fontSize: '0.85rem',
                            color: '#94a3b8'
                          }}>
                            {item.files.map((file, idx) => (
                              <div key={idx} style={{
                                padding: '4px 0',
                                borderBottom: idx < item.files!.length - 1 ? '1px solid rgba(255, 255, 255, 0.08)' : 'none'
                              }}>
                                {file}
                              </div>
                            ))}
                          </div>
                        </div>
                        )}
                          <div style={{ textAlign: 'center' }}>
                          {downloadReady ? (
                                  <button
                                      onClick={() => {
                                          if (downloadOpId) {
                                            BrowserOpenURL(`${getApiBaseUrl()}/api/download/${downloadOpId}`);
                                          }
                                      }}
                                      style={{
                                          padding: '8px 16px',
                                          background: 'linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%)',
                                          color: 'white',
                                          border: 'none',
                                          borderRadius: '6px',
                                          fontSize: '0.9rem',
                                          fontWeight: '500',
                                          cursor: 'pointer',
                                          boxShadow: '0 2px 4px rgba(14, 165, 233, 0.3)',
                                          transition: 'all 0.2s ease'
                                      }}
                                      onMouseEnter={(e) => {
                                          e.currentTarget.style.transform = 'translateY(-1px)';
                                          e.currentTarget.style.boxShadow = '0 4px 8px rgba(14, 165, 233, 0.4)';
                                      }}
                                      onMouseLeave={(e) => {
                                          e.currentTarget.style.transform = 'translateY(0)';
                                          e.currentTarget.style.boxShadow = '0 2px 4px rgba(14, 165, 233, 0.3)';
                                      }}
                                    >
                                      {isSingleFile ? `📥 Download ${singleFileThumb}` : '📥 Download Files'}
                                    </button>
                                  ) : hasErrorText ? (
                                  <div style={{
                                      display: 'inline-flex',
                                      alignItems: 'center',
                                      gap: '8px',
                                      padding: '8px 16px',
                                      background: 'rgba(239, 68, 68, 0.1)',
                                      border: '1px solid rgba(239, 68, 68, 0.3)',
                                      borderRadius: '6px',
                                      color: '#fca5a5',
                                      fontSize: '0.9rem'
                                  }}>
                                      ⚠️ {item.text}
                                  </div>
                              ) : (
                                    <div style={{
                                      display: 'inline-flex',
                                      alignItems: 'center',
                                      gap: '8px',
                                      padding: '8px 16px',
                                      background: 'rgba(255, 255, 255, 0.03)',
                                      borderRadius: '6px',
                                      color: '#94a3b8',
                                      fontSize: '0.9rem'
                                  }}>
                                      <div className="spinner" style={{
                                          width: '16px',
                                          height: '16px',
                                          border: '2px solid rgba(148, 163, 184, 0.3)',
                                          borderTop: '2px solid #0ea5e9',
                                          borderRadius: '50%',
                                          animation: 'spin 1s linear infinite'
                                      }}></div>
                                      {isSingleFile ? 'Preparing single file...' : 'Compressing files...'}
                                  </div>
                              )}
                          </div>
                      </div>
                  )}
              </div>

              {/* Subtle gradient overlay */}
              <div style={{
                  position: 'absolute',
                  top: 0,
                  right: 0,
                  width: '100px',
                  height: '100px',
                  background: 'linear-gradient(135deg, rgba(14, 165, 233, 0.05) 0%, transparent 70%)',
                  borderRadius: '0 12px 0 50px',
                  pointerEvents: 'none'
              }}></div>
          </div>
      );
  };

  const openInviteModal = async () => {
    setInviteOpen(true);
    setInviteLoading(true);
    setInviteError(null);
    try {
      const users = await httpFetchUsers();
      // Only users not already in the current room (by membership list or their roomId)
      const filtered = users.filter((u) => {
        if (u.id === currentUser.id) return false;
        if (currentRoom.userIds.includes(u.id)) return false;
        if (u.roomId && u.roomId === currentRoom.id) return false;
        return true;
      });
      setInviteUsers(filtered);
    } catch (err) {
      console.error('Failed to load users', err);
      setInviteError('Failed to load users');
    } finally {
      setInviteLoading(false);
    }
  };

  const handleInvite = async (userId: string, userName: string) => {
    try {
      if (appMode === 'client') {
        await httpInviteUser({ userId, inviterId: currentUser.id, message: `Join ${currentRoom.name}` });
      } else {
        await hostInviteUser(userId);
      }
      setInvitedIds((prev) => new Set(prev).add(userId));
    } catch (err) {
      console.error('Failed to invite user', err);
      setInviteError('Failed to invite user');
    }
  };

  return (
    <div className="room-shell">
      {/* Chat */}
      <div className="panel">
        <div className="panel-header">
          <div>
            <p className="pill" style={{ display: 'inline-block', marginBottom: '4px' }}>Chat</p>
            <h3 style={{ margin: 0 }}>{currentRoom.name}</h3>
          </div>
          <div style={{ display: 'flex', gap: '8px' }}>
            <button className="icon-btn" onClick={openInviteModal} title="Invite users">➕ Invite</button>
            <button className="secondary-btn" onClick={handleLeave}>Leave Room</button>
          </div>
        </div>
        <div className="chat-list">
          {messages.map(msg => (
            <div key={msg.id} className={`chat-bubble ${msg.userId === currentUser.id ? 'chat-bubble-me' : 'chat-bubble-other'}`}>
              {msg.userId === currentUser.id ? (
                <div>
                  <div className="chat-sender" style={{ textAlign: 'right' }}>You</div>
                  <div className="chat-message">{msg.message}</div>
                </div>
              ) : (
                <div>
                  <div className="chat-sender">{msg.userName}</div>
                  <div className="chat-message">{msg.message}</div>
                </div>
              )}
            </div>
          ))}
          <div ref={chatEndRef} />
        </div>
        <form onSubmit={handleSend} className="chat-input">
          <input 
            value={newMessage} 
            onChange={e => setNewMessage(e.target.value)}
            className="text-input"
            placeholder="Type a message..."
          />
          <button type="submit" className="primary-btn">Send</button>
        </form>
        <div className="muted" style={{ fontSize: '0.9rem' }}>
            Logged in as: <strong>{currentUser.name}</strong>
        </div>
      </div>

      {/* Clipboard */}
      <div className="panel">
        <div className="panel-header" style={{
          background: 'rgba(255, 255, 255, 0.02)',
          borderBottom: '1px solid rgba(255, 255, 255, 0.08)',
          padding: '16px 20px'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <div style={{
              width: '40px',
              height: '40px',
              borderRadius: '50%',
              background: 'linear-gradient(135deg, #0ea5e9 0%, #0284c7 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: '1.2rem',
              boxShadow: '0 4px 8px rgba(14, 165, 233, 0.3)'
            }}>
              📋
            </div>
            <div>
              <p className="pill" style={{
                display: 'inline-block',
                marginBottom: '4px',
                background: 'rgba(14, 165, 233, 0.12)',
                color: '#38bdf8',
                border: '1px solid rgba(14, 165, 233, 0.25)',
                padding: '4px 8px',
                borderRadius: '12px',
                fontSize: '0.75rem',
                fontWeight: '600'
              }}>Clipboard</p>
              <h3 style={{
                margin: 0,
                color: '#e2e8f0',
                fontSize: '1.25rem',
                fontWeight: '700'
              }}>Shared Items</h3>
              <div style={{
                fontSize: '0.8rem',
                color: '#94a3b8',
                marginTop: '2px'
              }}>
                {operations.length} item{operations.length !== 1 ? 's' : ''} shared
              </div>
            </div>
          </div>
        </div>
        <div className="clipboard-list" style={{
          padding: '20px',
          background: 'rgba(255, 255, 255, 0.01)',
          flex: 1,
          minHeight: 0
        }}>
          {operations.length === 0 ? (
             <div style={{
               textAlign: 'center',
               padding: '40px 20px',
               color: '#94a3b8'
             }}>
               <div style={{
                 fontSize: '3rem',
                 marginBottom: '16px',
                 opacity: 0.5
               }}>📋</div>
               <div style={{
                 fontSize: '1.1rem',
                 fontWeight: '600',
                 marginBottom: '8px',
                 color: '#e2e8f0'
               }}>No shared items yet</div>
               <div style={{
                 fontSize: '0.9rem'
               }}>Clipboard items will appear here when shared</div>
             </div>
          ) : (
              operations.slice().reverse().map(renderClipboardItem)
          )}
        </div>
      </div>
      <InviteModal
        open={inviteOpen}
        onClose={() => setInviteOpen(false)}
        users={inviteUsers}
        loading={inviteLoading}
        error={inviteError}
        invitedIds={invitedIds}
        currentRoomId={currentRoom.id}
        currentRoomUserIds={currentRoom.userIds}
        onInvite={handleInvite}
      />
    </div>
  );
};

const InviteModal: React.FC<{
  open: boolean;
  onClose: () => void;
  users: User[];
  loading: boolean;
  error: string | null;
  invitedIds: Set<string>;
  currentRoomId: string;
  currentRoomUserIds: string[];
  onInvite: (id: string, name: string) => void;
}> = ({ open, onClose, users, loading, error, invitedIds, currentRoomId, currentRoomUserIds, onInvite }) => {
  if (!open) return null;
  return (
    <div className="modal-backdrop" style={{ zIndex: 2000 }}>
      <div className="modal-card" style={{ maxWidth: '520px', width: '520px' }}>
        <div className="modal-head">
          <h3 style={{ margin: 0 }}>Invite Users</h3>
          <button className="modal-close" onClick={onClose}>✕</button>
        </div>
        <div style={{ marginBottom: '12px', color: '#94a3b8', fontSize: '0.9rem' }}>
          Select a user to invite to this room.
        </div>
        {loading && <div className="muted">Loading users...</div>}
        {error && <div style={{ color: '#fca5a5', marginBottom: '8px' }}>{error}</div>}
        {!loading && users.length === 0 && <div className="muted">No available users to invite.</div>}
        <div className="invite-list">
          {users.map((u) => {
            const invited = invitedIds.has(u.id);
            const inRoom = u.roomId === currentRoomId || currentRoomUserIds.includes(u.id);
            return (
              <div key={u.id} className="invite-row">
                <div>
                  <div className="invite-name">{u.name}</div>
                  <div className="invite-sub">{u.isOnline ? 'Online' : 'Offline'}</div>
                </div>
                <button
                  className="primary-btn"
                  style={{ padding: '8px 12px', minWidth: '90px', opacity: invited || inRoom ? 0.6 : 1 }}
                  disabled={invited || inRoom}
                  onClick={() => onInvite(u.id, u.name)}
                >
                  {inRoom ? 'In room' : invited ? 'Invited' : 'Invite'}
                </button>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default RoomView;
