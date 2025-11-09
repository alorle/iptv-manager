import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import './App.css'
import ChannelsList from './components/ChannelsList/ChannelsList';


const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ChannelsList />
    </QueryClientProvider>
  );
}

export default App
