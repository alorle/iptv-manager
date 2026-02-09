import { Routes, Route, Link } from "react-router-dom";

function Home() {
  return (
    <div>
      <h1 className="text-2xl font-bold">IPTV Manager</h1>
      <p className="mt-2 text-gray-600">Dashboard coming soon.</p>
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
        <Link to="/about" className="text-blue-600 hover:underline">About</Link>
      </nav>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/about" element={<About />} />
      </Routes>
    </div>
  );
}
