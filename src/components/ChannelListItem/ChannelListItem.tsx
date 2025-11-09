import { Trash2 } from 'lucide-react';
import { components } from '../../lib/api/v1';
import DeleteChannelDialog from '../DeleteChannelDialog/DeleteChannelDialog';
import ChannelFormDialog from '../ChannelFormDialog/ChannelFormDialog';
import StreamFormDialog from '../StreamFormDialog/StreamFormDialog';
import { Button } from '@/components/ui/button';

type Channel = components["schemas"]["Channel"];
type Stream = components["schemas"]["Stream"];

interface ChannelFormData {
  title: string;
  guide_id: string;
  logo: string;
  group_title: string;
}

interface ChannelListItemProps {
  index: number;
  channel: Channel;
  onDelete: (id: string) => Promise<void>;
  onUpdate: (id: string, data: ChannelFormData) => Promise<void>;
  onUpdateStreams: (channelId: string, streams: Stream[]) => Promise<void>;
}

export default function ChannelListItem({
  index,
  channel,
  onDelete,
  onUpdate,
  onUpdateStreams,
}: ChannelListItemProps) {
  const handleUpdate = async (data: ChannelFormData) => {
    await onUpdate(channel.id, data);
  };

  const handleAddStream = async (streamData: Omit<Stream, 'id' | 'channel_id'>) => {
    const newStream: Stream = {
      id: '00000000-0000-0000-0000-000000000000', // Backend generates
      channel_id: channel.id,
      ...streamData,
    };
    await onUpdateStreams(channel.id, [...channel.streams, newStream]);
  };

  const handleEditStream = async (
    streamIndex: number,
    streamData: Omit<Stream, 'id' | 'channel_id'>
  ) => {
    const updatedStreams = channel.streams.map((s, i) =>
      i === streamIndex
        ? { ...s, ...streamData }
        : s
    );
    await onUpdateStreams(channel.id, updatedStreams);
  };

  const handleDeleteStream = async (streamIndex: number) => {
    const updatedStreams = channel.streams.filter((_, i) => i !== streamIndex);
    await onUpdateStreams(channel.id, updatedStreams);
  };

  return (
    <div className="mb-6 p-4 border border-gray-300 rounded-lg bg-white shadow-sm">
      <div className="flex justify-between items-start mb-3">
        <div>
          <h2 className="text-xl font-bold text-gray-900">
            #{index} - {channel.title}
          </h2>
          <p className="text-sm text-gray-600 mt-1">
            <span className="font-semibold">Guide ID:</span> {channel.guide_id}
          </p>
          {channel.logo && (
            <p className="text-sm text-gray-600">
              <span className="font-semibold">Logo:</span> {channel.logo}
            </p>
          )}
          <p className="text-sm text-gray-600">
            <span className="font-semibold">Group:</span> {channel.group_title}
          </p>
        </div>
        <div className="flex gap-2">
          <ChannelFormDialog
            mode="edit"
            channel={channel}
            onSubmit={handleUpdate}
          />
          <DeleteChannelDialog
            channelId={channel.id}
            channelTitle={channel.title}
            onDelete={onDelete}
          />
        </div>
      </div>

      <div className="flex justify-between items-center mt-4 mb-2">
        <h3 className="text-lg font-semibold text-gray-800">
          Streams ({channel.streams.length})
        </h3>
        <StreamFormDialog mode="create" onSubmit={handleAddStream} />
      </div>

      <div className="ml-4 space-y-3">
        {channel.streams.length === 0 ? (
          <p className="text-sm text-gray-500 italic">
            No streams yet. Add one to get started.
          </p>
        ) : (
          channel.streams.map((stream: Stream, streamIndex: number) => (
            <div
              key={stream.id}
              className="p-3 bg-gray-50 rounded border border-gray-200"
            >
              <div className="flex justify-between items-start mb-2">
                <p className="font-semibold text-gray-700">
                  Stream #{streamIndex + 1}
                </p>
                <div className="flex gap-1">
                  <StreamFormDialog
                    mode="edit"
                    stream={stream}
                    onSubmit={(data) => handleEditStream(streamIndex, data)}
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDeleteStream(streamIndex)}
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              </div>
              <p className="text-sm font-mono bg-gray-100 p-1 rounded mb-1">
                {stream.acestream_id}
              </p>
              {stream.quality && (
                <p className="text-sm text-gray-600">
                  <span className="font-semibold">Quality:</span> {stream.quality}
                </p>
              )}
              {stream.tags && stream.tags.length > 0 && (
                <p className="text-sm text-gray-600">
                  <span className="font-semibold">Tags:</span> {stream.tags.join(', ')}
                </p>
              )}
              <p className="text-sm text-gray-600">
                <span className="font-semibold">Network Caching:</span>{' '}
                {stream.network_caching}ms
              </p>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
