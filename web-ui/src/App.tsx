import { useState } from 'react';
import { ChannelList } from './components/ChannelList';
import { EditOverrideForm } from './components/EditOverrideForm';
import { ToastContainer } from './components/ToastContainer';
import { useToast } from './hooks/useToast';
import type { Channel } from './types';
import './App.css';

function App() {
  const [selectedChannel, setSelectedChannel] = useState<Channel | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const toast = useToast();

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
        toast={toast}
      />
      {selectedChannel && (
        <EditOverrideForm
          channel={selectedChannel}
          onClose={handleFormClose}
          onSave={handleFormSave}
          toast={toast}
        />
      )}
      <ToastContainer toasts={toast.toasts} onClose={toast.closeToast} />
    </div>
  );
}

export default App;
