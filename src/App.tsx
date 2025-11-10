import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import HealthCheck from "./components/Health";
import { ThemeToggle } from "./components/ThemeToggle";

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <HealthCheck />
      <ThemeToggle />
    </QueryClientProvider>
  );
}

export default App;
