import React, { useState, useEffect, useRef } from 'react';

const WebSocketClient = () => {
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [roomCode, setRoomCode] = useState('');
  const [connected, setConnected] = useState(false);
  const wsRef = useRef(null);

  const connectToRoom = () => {
    if (!roomCode) return;
    
    const ws = new WebSocket(`ws://${window.location.host}/join-room/${roomCode}`);
    
    ws.onopen = () => {
      setConnected(true);
      setMessages(prev => [...prev, 'Connected to room ' + roomCode]);
    };

    ws.onmessage = (event) => {
      setMessages(prev => [...prev, event.data]);
    };

    ws.onclose = () => {
      setConnected(false);
      setMessages(prev => [...prev, 'Disconnected from room']);
    };

    wsRef.current = ws;
  };

  const sendMessage = () => {
    if (!inputMessage || !wsRef.current) return;
    wsRef.current.send(inputMessage);
    setMessages(prev => [...prev, '→ ' + inputMessage]);
    setInputMessage('');
  };

  const createRoom = async () => {
    try {
      const response = await fetch('/create-room', { method: 'POST' });
      if (!response.ok) throw new Error('Failed to create room');
      const data = await response.json();
      setRoomCode(data.code);
    } catch (error) {
      console.error('Error creating room:', error);
    }
  };

  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  return (
    <div style={{ maxWidth: '500px', margin: '20px auto', padding: '20px' }}>
      {!connected ? (
        <div>
          <input
            type="text"
            placeholder="Enter room code"
            value={roomCode}
            onChange={(e) => setRoomCode(e.target.value)}
            style={{ marginRight: '10px', padding: '5px' }}
          />
          <button onClick={connectToRoom} style={{ marginRight: '10px', padding: '5px' }}>
            Join Room
          </button>
          <button onClick={createRoom} style={{ padding: '5px' }}>
            Create Room
          </button>
        </div>
      ) : (
        <div>
          <div
            style={{
              height: '300px',
              overflowY: 'auto',
              border: '1px solid #ccc',
              marginBottom: '10px',
              padding: '10px'
            }}
          >
            {messages.map((msg, idx) => (
              <div key={idx} style={{ marginBottom: '5px' }}>
                {msg}
              </div>
            ))}
          </div>
          <div style={{ display: 'flex' }}>
            <input
              type="text"
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
              placeholder="Type a message..."
              style={{ flexGrow: 1, marginRight: '10px', padding: '5px' }}
            />
            <button onClick={sendMessage} style={{ padding: '5px' }}>
              Send
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default WebSocketClient;