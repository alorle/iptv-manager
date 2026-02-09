import { useEffect, useState } from "react";

interface Stream {
  info_hash: string;
  channel_name: string;
}

export default function Streams() {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStreams = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch("/api/streams");

      if (!response.ok) {
        throw new Error(`Failed to fetch streams: ${response.status}`);
      }

      const data = await response.json();
      setStreams(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStreams();
  }, []);

  if (loading) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Streams</h1>
        <p className="text-gray-600">Loading streams...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Streams</h1>
        <p className="text-red-600">Error: {error}</p>
      </div>
    );
  }

  const renderContent = () => {
    if (streams.length === 0) {
      return (
        <p className="text-gray-600">No streams found.</p>
      );
    }

    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 border border-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Info Hash
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Channel Name
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {streams.map((stream) => (
              <tr key={stream.info_hash} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {stream.info_hash}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {stream.channel_name}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Streams</h1>
      {renderContent()}
    </div>
  );
}
