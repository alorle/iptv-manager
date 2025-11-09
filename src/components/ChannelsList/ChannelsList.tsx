import { paths, components } from '../../lib/api/v1';
import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";
import { useQueryClient, useMutation } from '@tanstack/react-query';
import ChannelListItem from '../ChannelListItem/ChannelListItem';
import StreamFormDialog from '../StreamFormDialog/StreamFormDialog';

const fetchClient = createFetchClient<paths>({
  baseUrl: "/api/",
});
const $api = createClient(fetchClient);

type Stream = components["schemas"]["Stream"];

function ChannelsList() {
  const queryClient = useQueryClient();

  const { data: channels, error, isLoading } = $api.useQuery(
    "get",
    "/channels",
  );

  // Stream CRUD mutations
  const createStreamMutation = useMutation({
    mutationFn: async (data: Omit<Stream, 'id'>) => {
      const response = await fetch('/api/streams', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to create stream');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const updateStreamMutation = useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Omit<Stream, 'id'> }) => {
      const response = await fetch(`/api/streams/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(data),
      });
      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to update stream');
      }
      return response.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const deleteStreamMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await fetch(`/api/streams/${id}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error('Failed to delete stream');
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['get', '/channels'],
      });
    },
  });

  const handleCreateStream = async (data: Omit<Stream, 'id'>) => {
    await createStreamMutation.mutateAsync(data);
  };

  const handleUpdateStream = async (id: string, data: Omit<Stream, 'id'>) => {
    await updateStreamMutation.mutateAsync({ id, data });
  };

  const handleDeleteStream = async (id: string) => {
    await deleteStreamMutation.mutateAsync(id);
  };

  if (isLoading || !channels) return <div className="p-4">Loading...</div>;

  if (error) return <div className="p-4 text-red-600">An error occurred: {error.message}</div>;

  return (
    <div className="max-w-4xl mx-auto p-4">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Channels</h1>
        <StreamFormDialog mode="create" onSubmit={handleCreateStream} />
      </div>

      {channels.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <p>No channels yet. Add streams to create channels!</p>
        </div>
      ) : (
        channels.map((channel, index) => (
          <ChannelListItem
            key={channel.guide_id}
            channel={channel}
            index={index + 1}
            onCreateStream={handleCreateStream}
            onUpdateStream={handleUpdateStream}
            onDeleteStream={handleDeleteStream}
          />
        ))
      )}
    </div>
  );
}

export default ChannelsList;
