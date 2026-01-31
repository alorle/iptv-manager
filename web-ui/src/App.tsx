import { useState } from 'react';
import { ChannelList } from './components/ChannelList';
import { EditOverrideForm } from './components/EditOverrideForm';
import type { Channel } from './types';
import './App.css';

function App() {
  const [selectedChannel, setSelectedChannel] = useState<Channel | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const handleChannelSelect = (channel: Channel) => {
    setSelectedChannel(channel);
  };

  const handleFormClose = () => {
    setSelectedChannel(null);
  };

  const handleFormSave = () => {
    setSelectedChannel(null);
    setRefreshTrigger((prev) => prev + 1);
  };

  return (
    <div className="app">
      <ChannelList
        onChannelSelect={handleChannelSelect}
        refreshTrigger={refreshTrigger}
      />
      {selectedChannel && (
        <EditOverrideForm
          channel={selectedChannel}
          onClose={handleFormClose}
          onSave={handleFormSave}
        />
      )}
    </div>
  );
}

export default App;
