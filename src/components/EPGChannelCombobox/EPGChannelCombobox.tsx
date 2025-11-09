import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Check, ChevronsUpDown, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import createFetchClient from 'openapi-fetch';
import type { paths } from '@/lib/api/v1';

const client = createFetchClient<paths>({ baseUrl: '/api' });

interface EPGChannel {
  id: string;
  name: string;
  logo?: string;
}

interface EPGChannelComboboxProps {
  value: string;
  onChange: (value: string, logo?: string) => void;
  error?: string;
}

export default function EPGChannelCombobox({
  value,
  onChange,
  error,
}: EPGChannelComboboxProps) {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const { data: epgChannels, isLoading, isError } = useQuery({
    queryKey: ['epg', 'channels', searchQuery],
    queryFn: async () => {
      const response = await client.GET('/epg/channels', {
        params: { query: { search: searchQuery || undefined } },
      });
      if (response.error) {
        throw new Error('Failed to fetch EPG channels');
      }
      return response.data as EPGChannel[];
    },
    staleTime: 1000 * 60 * 60, // Cache for 1 hour
  });

  const selectedChannel = epgChannels?.find((ch) => ch.id === value);

  const handleSelect = (channel: EPGChannel) => {
    onChange(channel.id, channel.logo);
    setOpen(false);
    setSearchQuery('');
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    if (!open) {
      setSearchQuery('');
    }
  }, [open]);

  return (
    <div className="relative">
      <Button
        type="button"
        variant="outline"
        role="combobox"
        aria-expanded={open}
        className={`w-full justify-between ${error ? 'border-red-500' : ''}`}
        onClick={() => setOpen(!open)}
      >
        {selectedChannel ? (
          <div className="flex items-center gap-2">
            {selectedChannel.logo && (
              <img
                src={selectedChannel.logo}
                alt={selectedChannel.name}
                className="h-5 w-5 object-contain"
              />
            )}
            <span className="truncate">
              {selectedChannel.name} ({selectedChannel.id})
            </span>
          </div>
        ) : (
          <span className="text-muted-foreground">Select EPG channel...</span>
        )}
        <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </Button>

      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-md border bg-white shadow-lg">
          <div className="p-2 border-b">
            <Input
              placeholder="Search channels..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-9"
              autoFocus
            />
          </div>

          <div className="max-h-[300px] overflow-y-auto p-1">
            {isLoading && (
              <div className="flex items-center justify-center py-6">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            )}

            {isError && (
              <div className="py-6 text-center text-sm text-red-600">
                Failed to load EPG channels. Please check EPG_URL configuration.
              </div>
            )}

            {!isLoading && !isError && epgChannels && epgChannels.length === 0 && (
              <div className="py-6 text-center text-sm text-gray-500">
                No channels found.
              </div>
            )}

            {!isLoading &&
              !isError &&
              epgChannels &&
              epgChannels.map((channel) => (
                <button
                  key={channel.id}
                  type="button"
                  className="relative flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-2 text-sm outline-none hover:bg-gray-100 focus:bg-gray-100"
                  onClick={() => handleSelect(channel)}
                >
                  {channel.logo && (
                    <img
                      src={channel.logo}
                      alt={channel.name}
                      className="h-6 w-6 object-contain"
                    />
                  )}
                  <div className="flex flex-col items-start flex-1 min-w-0">
                    <span className="font-medium truncate w-full">
                      {channel.name}
                    </span>
                    <span className="text-xs text-gray-500 truncate w-full">
                      {channel.id}
                    </span>
                  </div>
                  {value === channel.id && (
                    <Check className="h-4 w-4 shrink-0" />
                  )}
                </button>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}
