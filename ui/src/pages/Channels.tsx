import { useEffect, useState } from "react";

interface Channel {
  name: string;
}

interface Stream {
  info_hash: string;
  channel_name: string;
}

export default function Channels() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [streams, setStreams] = useState<Stream[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoading(true);
        setError(null);

        const [channelsRes, streamsRes] = await Promise.all([
          fetch("/api/channels"),
          fetch("/api/streams"),
        ]);

        if (!channelsRes.ok) {
          throw new Error(`Failed to fetch channels: ${channelsRes.status}`);
        }

        if (!streamsRes.ok) {
          throw new Error(`Failed to fetch streams: ${streamsRes.status}`);
        }

        const channelsData = await channelsRes.json();
        const streamsData = await streamsRes.json();

        setChannels(channelsData);
        setStreams(streamsData);
      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  const getStreamCount = (channelName: string): number => {
    return streams.filter((s) => s.channel_name === channelName).length;
  };

  if (loading) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Channels</h1>
        <p className="text-gray-600">Loading channels...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Channels</h1>
        <p className="text-red-600">Error: {error}</p>
      </div>
    );
  }

  if (channels.length === 0) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Channels</h1>
        <p className="text-gray-600">No channels found. Create your first channel to get started.</p>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4">Channels</h1>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 border border-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Stream Count
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {channels.map((channel) => (
              <tr key={channel.name} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {channel.name}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {getStreamCount(channel.name)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
