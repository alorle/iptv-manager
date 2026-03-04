import { Routes, Route, Link } from "react-router-dom";
import Channels from "./pages/Channels";
import Streams from "./pages/Streams";
import EPGSubscriptions from "./pages/EPGSubscriptions";
import EPGMappingAdmin from "./pages/EPGMappingAdmin";

export default function App() {
  return (
    <div className="mx-auto max-w-4xl p-8">
      <nav className="mb-8 flex gap-4">
        <Link to="/" className="text-blue-600 hover:underline">Channels</Link>
        <Link to="/streams" className="text-blue-600 hover:underline">Streams</Link>
        <Link to="/epg-subscriptions" className="text-blue-600 hover:underline">EPG Subscriptions</Link>
        <Link to="/epg-mapping-admin" className="text-blue-600 hover:underline">EPG Mapping Admin</Link>
      </nav>
      <Routes>
        <Route path="/" element={<Channels />} />
        <Route path="/streams" element={<Streams />} />
        <Route path="/epg-subscriptions" element={<EPGSubscriptions />} />
        <Route path="/epg-mapping-admin" element={<EPGMappingAdmin />} />
      </Routes>
    </div>
  );
}
