import { Trash2 } from 'lucide-react';
import { components } from '../../lib/api/v1';
import StreamFormDialog from '../StreamFormDialog/StreamFormDialog';
import { Button } from '@/components/ui/button';

type Channel = components["schemas"]["Channel"];
type Stream = components["schemas"]["Stream"];

interface ChannelListItemProps {
  index: number;
  channel: Channel;
  onCreateStream: (data: Omit<Stream, 'id'>) => Promise<void>;
  onUpdateStream: (id: string, data: Omit<Stream, 'id'>) => Promise<void>;
  onDeleteStream: (id: string) => Promise<void>;
}

export default function ChannelListItem({
  index,
  channel,
  onCreateStream,
  onUpdateStream,
  onDeleteStream,
}: ChannelListItemProps) {
  const handleAddStream = async (streamData: Omit<Stream, 'id'>) => {
    // Include the channel's guide_id when creating a new stream
    await onCreateStream({
      ...streamData,
      guide_id: channel.guide_id,
    });
  };

  const handleEditStream = async (stream: Stream, streamData: Omit<Stream, 'id'>) => {
    await onUpdateStream(stream.id!, streamData);
  };

  const handleDeleteStream = async (streamId: string) => {
    await onDeleteStream(streamId);
  };

  return (
    <div className="mb-6 p-4 border border-gray-300 rounded-lg bg-white shadow-sm">
      <div className="flex items-start gap-4 mb-3">
        {channel.logo && (
          <img
            src={channel.logo}
            alt={channel.title || 'Channel logo'}
            className="w-16 h-16 object-contain rounded"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
            }}
          />
        )}
        <div className="flex-1">
          <h2 className="text-xl font-bold text-gray-900">
            #{index} - {channel.title}
          </h2>
          <p className="text-sm text-gray-600 mt-1">
            <span className="font-semibold">Guide ID:</span> {channel.guide_id}
          </p>
          <p className="text-sm text-gray-600">
            <span className="font-semibold">Group:</span> {channel.group_title}
          </p>
        </div>
      </div>

      <div className="flex justify-between items-center mt-4 mb-2">
        <h3 className="text-lg font-semibold text-gray-800">
          Streams ({channel.streams?.length || 0})
        </h3>
        <StreamFormDialog
          mode="create"
          onSubmit={handleAddStream}
          guideId={channel.guide_id}
        />
      </div>

      <div className="ml-4 space-y-3">
        {!channel.streams || channel.streams.length === 0 ? (
          <p className="text-sm text-gray-500 italic">
            No streams yet. Add one to get started.
          </p>
        ) : (
          channel.streams.map((stream: Stream, streamIndex: number) => (
            <div
              key={stream.id}
              className="p-3 bg-gray-50 rounded border border-gray-200"
            >
              <div className="flex justify-between items-start gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-2">
                    <p className="font-semibold text-gray-700">
                      Stream #{streamIndex + 1}
                    </p>
                    {stream.quality && (
                      <span className="px-2 py-0.5 text-xs font-medium bg-blue-100 text-blue-800 rounded">
                        {stream.quality}
                      </span>
                    )}
                    {stream.tags && stream.tags.length > 0 && (
                      <div className="flex gap-1">
                        {stream.tags.map((tag, idx) => (
                          <span
                            key={idx}
                            className="px-2 py-0.5 text-xs font-medium bg-gray-200 text-gray-700 rounded"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                  <p className="text-sm font-mono bg-white p-2 rounded border border-gray-200 break-all">
                    {stream.acestream_id}
                  </p>
                </div>
                <div className="flex gap-1 flex-shrink-0">
                  <StreamFormDialog
                    mode="edit"
                    stream={stream}
                    onSubmit={(data) => handleEditStream(stream, data)}
                    guideId={channel.guide_id}
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDeleteStream(stream.id!)}
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
