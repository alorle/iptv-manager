import { paths, components } from '../../lib/api/v1';
import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";
import { useQueryClient, useMutation } from '@tanstack/react-query';
import ChannelListItem from '../ChannelListItem/ChannelListItem';
import ChannelFormDialog from '../ChannelFormDialog/ChannelFormDialog';

const fetchClient = createFetchClient<paths>({
  baseUrl: "/api/",
});
const $api = createClient(fetchClient);

type Stream = components["schemas"]["Stream"];

interface ChannelFormData {
  title: string;
  guide_id: string;
  logo: string;
  group_title: string;
}

function ChannelsList() {
  const queryClient = useQueryClient();

  const { data: channels, error, isLoading } = $api.useQuery(
    "get",
    "/channels",
  );

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/channels/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error('Failed to delete channel');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: ChannelFormData) => {
      const response = await fetch('/api/channels', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          id: '00000000-0000-0000-0000-000000000000', // Backend will generate
          ...data,
          streams: [], // Start with no streams
        }),
      });
      if (!response.ok) {
        throw new Error('Failed to create channel');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: string; data: ChannelFormData }) => {
      // First get the existing channel to preserve streams
      const existingChannel = channels?.find(ch => ch.id === id);
      if (!existingChannel) {
        throw new Error('Channel not found');
      }

      const response = await fetch(`/api/channels/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          id,
          ...data,
          streams: existingChannel.streams, // Preserve existing streams
        }),
      });
      if (!response.ok) {
        throw new Error('Failed to update channel');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const updateStreamsMutation = useMutation({
    mutationFn: async ({ channelId, streams }: { channelId: string; streams: Stream[] }) => {
      const channel = channels?.find(ch => ch.id === channelId);
      if (!channel) {
        throw new Error('Channel not found');
      }

      const response = await fetch(`/api/channels/${channelId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...channel,
          streams,
        }),
      });
      if (!response.ok) {
        throw new Error('Failed to update streams');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const handleDelete = async (id: string) => {
    await deleteMutation.mutateAsync(id);
  };

  const handleCreate = async (data: ChannelFormData) => {
    await createMutation.mutateAsync(data);
  };

  const handleUpdate = async (id: string, data: ChannelFormData) => {
    await updateMutation.mutateAsync({ id, data });
  };

  const handleUpdateStreams = async (channelId: string, streams: Stream[]) => {
    await updateStreamsMutation.mutateAsync({ channelId, streams });
  };

  if (isLoading || !channels) return <div className="p-4">Loading...</div>;

  if (error) return <div className="p-4 text-red-600">An error occurred: {error.message}</div>;

  return (
    <div className="max-w-4xl mx-auto p-4">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Channels</h1>
        <ChannelFormDialog mode="create" onSubmit={handleCreate} />
      </div>

      {channels.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <p>No channels yet. Create your first channel to get started!</p>
        </div>
      ) : (
        channels.map((channel, index) => (
          <ChannelListItem
            key={channel.id}
            channel={channel}
            index={index + 1}
            onDelete={handleDelete}
            onUpdate={handleUpdate}
            onUpdateStreams={handleUpdateStreams}
          />
        ))
      )}
    </div>
  );
}

export default ChannelsList;
