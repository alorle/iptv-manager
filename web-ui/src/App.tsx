import { useState } from 'react';
import { ChannelList } from './components/ChannelList';
import type { Channel } from './types';
import './App.css';

function App() {
  const [_selectedChannel, setSelectedChannel] = useState<Channel | null>(null);

  const handleChannelSelect = (channel: Channel) => {
    setSelectedChannel(channel);
    console.log('Selected channel:', channel);
    // TODO: Open edit panel in future user story
  };

  return (
    <div className="app">
      <ChannelList onChannelSelect={handleChannelSelect} />
    </div>
  );
}

export default App;
