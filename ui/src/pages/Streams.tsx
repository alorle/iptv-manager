import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";

interface Stream {
  info_hash: string;
  channel_name: string;
}

interface Channel {
  name: string;
}

export default function Streams() {
  const [streams, setStreams] = useState<Stream[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [infoHash, setInfoHash] = useState("");
  const [channelName, setChannelName] = useState("");
  const [searchTerm, setSearchTerm] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const fetchStreams = async () => {
    try {
      setLoading(true);
      setError(null);

      const [streamsRes, channelsRes] = await Promise.all([
        fetch("/api/streams"),
        fetch("/api/channels"),
      ]);

      if (!streamsRes.ok) {
        throw new Error(`Failed to fetch streams: ${streamsRes.status}`);
      }

      if (!channelsRes.ok) {
        throw new Error(`Failed to fetch channels: ${channelsRes.status}`);
      }

      const streamsData = await streamsRes.json();
      const channelsData = await channelsRes.json();

      setStreams(streamsData);
      setChannels(channelsData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStreams();
  }, []);

  const handleCreateStream = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!infoHash.trim()) {
      setFormError("Info Hash is required");
      return;
    }

    if (!channelName.trim()) {
      setFormError("Channel is required");
      return;
    }

    setSubmitting(true);

    try {
      const response = await fetch("/api/streams", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ info_hash: infoHash, channel_name: channelName }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        if (response.status === 409) {
          setFormError(errorData.error || "Stream already exists");
        } else if (response.status === 400) {
          setFormError(errorData.error || "Invalid stream data");
        } else {
          setFormError("Failed to create stream");
        }
        return;
      }

      toast.success("Stream created successfully");
      setDialogOpen(false);
      setInfoHash("");
      setChannelName("");
      setSearchTerm("");
      await fetchStreams();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDialogOpenChange = (open: boolean) => {
    setDialogOpen(open);
    if (!open) {
      setInfoHash("");
      setChannelName("");
      setSearchTerm("");
      setFormError(null);
    }
  };

  const filteredChannels = channels.filter((channel) =>
    channel.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

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
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Streams</h1>
        <Dialog open={dialogOpen} onOpenChange={handleDialogOpenChange}>
          <DialogTrigger asChild>
            <Button>New stream</Button>
          </DialogTrigger>
          <DialogContent>
            <form onSubmit={handleCreateStream}>
              <DialogHeader>
                <DialogTitle>Create stream</DialogTitle>
                <DialogDescription>
                  Add a new stream to an existing channel.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="info-hash">Info Hash</Label>
                  <Input
                    id="info-hash"
                    value={infoHash}
                    onChange={(e) => setInfoHash(e.target.value)}
                    placeholder="Enter stream info hash"
                    disabled={submitting}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="channel-search">Channel</Label>
                  <Input
                    id="channel-search"
                    value={searchTerm}
                    onChange={(e) => {
                      setSearchTerm(e.target.value);
                      setChannelName("");
                    }}
                    placeholder="Search channels..."
                    disabled={submitting}
                  />
                  {searchTerm && filteredChannels.length > 0 && (
                    <div className="border rounded-md max-h-48 overflow-y-auto">
                      {filteredChannels.map((channel) => (
                        <button
                          key={channel.name}
                          type="button"
                          className={`w-full text-left px-4 py-2 hover:bg-gray-100 ${
                            channelName === channel.name ? "bg-gray-100" : ""
                          }`}
                          onClick={() => {
                            setChannelName(channel.name);
                            setSearchTerm(channel.name);
                          }}
                          disabled={submitting}
                        >
                          {channel.name}
                        </button>
                      ))}
                    </div>
                  )}
                  {channelName && !searchTerm && (
                    <p className="text-sm text-gray-600">Selected: {channelName}</p>
                  )}
                </div>
                {formError && (
                  <p className="text-sm text-red-600">{formError}</p>
                )}
              </div>
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setDialogOpen(false)}
                  disabled={submitting}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={submitting}>
                  {submitting ? "Creating..." : "Create"}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
      {renderContent()}
    </div>
  );
}
