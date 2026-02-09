import { Routes, Route, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import Channels from "./pages/Channels";
import Streams from "./pages/Streams";
import EPGSubscriptions from "./pages/EPGSubscriptions";
import EPGMappingAdmin from "./pages/EPGMappingAdmin";

function Home() {
  return (
    <div>
      <h1 className="text-2xl font-bold">IPTV Manager</h1>
      <p className="mt-2 text-gray-600">Dashboard coming soon.</p>
      <div className="mt-4 flex gap-2">
        <Button>Default Button</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="outline">Outline</Button>
      </div>
    </div>
  );
}

function About() {
  return (
    <div>
      <h1 className="text-2xl font-bold">About</h1>
      <p className="mt-2 text-gray-600">IPTV Manager — manage playlists, channels and EPG data.</p>
    </div>
  );
}

export default function App() {
  return (
    <div className="mx-auto max-w-4xl p-8">
      <nav className="mb-8 flex gap-4">
        <Link to="/" className="text-blue-600 hover:underline">Home</Link>
        <Link to="/channels" className="text-blue-600 hover:underline">Channels</Link>
        <Link to="/streams" className="text-blue-600 hover:underline">Streams</Link>
        <Link to="/epg-subscriptions" className="text-blue-600 hover:underline">EPG Subscriptions</Link>
        <Link to="/epg-mapping-admin" className="text-blue-600 hover:underline">EPG Mapping Admin</Link>
        <Link to="/about" className="text-blue-600 hover:underline">About</Link>
      </nav>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/channels" element={<Channels />} />
        <Route path="/streams" element={<Streams />} />
        <Route path="/epg-subscriptions" element={<EPGSubscriptions />} />
        <Route path="/epg-mapping-admin" element={<EPGMappingAdmin />} />
        <Route path="/about" element={<About />} />
      </Routes>
    </div>
  );
}
