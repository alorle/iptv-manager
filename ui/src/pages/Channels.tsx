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
  const [dialogOpen, setDialogOpen] = useState(false);
  const [channelName, setChannelName] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

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

  useEffect(() => {
    fetchData();
  }, []);

  const getStreamCount = (channelName: string): number => {
    return streams.filter((s) => s.channel_name === channelName).length;
  };

  const handleCreateChannel = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!channelName.trim()) {
      setFormError("Channel name is required");
      return;
    }

    setSubmitting(true);

    try {
      const response = await fetch("/api/channels", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ name: channelName }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        if (response.status === 409) {
          setFormError(errorData.error || "Channel already exists");
        } else if (response.status === 400) {
          setFormError(errorData.error || "Invalid channel name");
        } else {
          setFormError("Failed to create channel");
        }
        return;
      }

      setDialogOpen(false);
      setChannelName("");
      await fetchData();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setSubmitting(false);
    }
  };

  const handleDialogOpenChange = (open: boolean) => {
    setDialogOpen(open);
    if (!open) {
      setChannelName("");
      setFormError(null);
    }
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

  const renderContent = () => {
    if (channels.length === 0) {
      return (
        <p className="text-gray-600">No channels found. Create your first channel to get started.</p>
      );
    }

    return (
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
    );
  };

  return (
    <div>
      <div className="flex justify-between items-center mb-4">
        <h1 className="text-2xl font-bold">Channels</h1>
        <Dialog open={dialogOpen} onOpenChange={handleDialogOpenChange}>
          <DialogTrigger asChild>
            <Button>New channel</Button>
          </DialogTrigger>
          <DialogContent>
            <form onSubmit={handleCreateChannel}>
              <DialogHeader>
                <DialogTitle>Create channel</DialogTitle>
                <DialogDescription>
                  Add a new channel to organize your streams.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    value={channelName}
                    onChange={(e) => setChannelName(e.target.value)}
                    placeholder="Enter channel name"
                    disabled={submitting}
                  />
                  {formError && (
                    <p className="text-sm text-red-600">{formError}</p>
                  )}
                </div>
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
