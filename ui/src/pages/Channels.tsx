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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { Trash2, Plus } from "lucide-react";

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
  const [sheetOpen, setSheetOpen] = useState(false);
  const [selectedChannel, setSelectedChannel] = useState<string | null>(null);
  const [streamDialogOpen, setStreamDialogOpen] = useState(false);
  const [streamInfoHash, setStreamInfoHash] = useState("");
  const [streamSubmitting, setStreamSubmitting] = useState(false);
  const [streamFormError, setStreamFormError] = useState<string | null>(null);

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

  const handleDeleteChannel = async (name: string) => {
    if (!confirm(`Are you sure you want to delete channel "${name}"? This will also delete all associated streams.`)) {
      return;
    }

    try {
      const response = await fetch(`/api/channels/${encodeURIComponent(name)}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: "Failed to delete channel" }));
        if (response.status === 404) {
          toast.error(errorData.error || "Channel not found");
        } else {
          toast.error(errorData.error || "Failed to delete channel");
        }
        return;
      }

      toast.success(`Channel "${name}" deleted successfully`);
      await fetchData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "An error occurred");
    }
  };

  const handleChannelClick = (channelName: string) => {
    setSelectedChannel(channelName);
    setSheetOpen(true);
  };

  const handleDeleteStream = async (infoHash: string) => {
    if (!confirm(`Are you sure you want to delete stream "${infoHash}"?`)) {
      return;
    }

    try {
      const response = await fetch(`/api/streams/${encodeURIComponent(infoHash)}`, {
        method: "DELETE",
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: "Failed to delete stream" }));
        if (response.status === 404) {
          toast.error(errorData.error || "Stream not found");
        } else {
          toast.error(errorData.error || "Failed to delete stream");
        }
        return;
      }

      toast.success(`Stream "${infoHash}" deleted successfully`);
      await fetchData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "An error occurred");
    }
  };

  const getChannelStreams = (channelName: string): Stream[] => {
    return streams.filter((s) => s.channel_name === channelName);
  };

  const handleCreateStream = async (e: React.FormEvent) => {
    e.preventDefault();
    setStreamFormError(null);

    if (!streamInfoHash.trim()) {
      setStreamFormError("Info Hash is required");
      return;
    }

    if (!selectedChannel) {
      setStreamFormError("No channel selected");
      return;
    }

    setStreamSubmitting(true);

    try {
      const response = await fetch("/api/streams", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ info_hash: streamInfoHash, channel_name: selectedChannel }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        if (response.status === 409) {
          setStreamFormError(errorData.error || "Stream already exists");
        } else if (response.status === 400) {
          setStreamFormError(errorData.error || "Invalid stream data");
        } else {
          setStreamFormError("Failed to create stream");
        }
        return;
      }

      toast.success("Stream created successfully");
      setStreamDialogOpen(false);
      setStreamInfoHash("");
      await fetchData();
    } catch (err) {
      setStreamFormError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setStreamSubmitting(false);
    }
  };

  const handleStreamDialogOpenChange = (open: boolean) => {
    setStreamDialogOpen(open);
    if (!open) {
      setStreamInfoHash("");
      setStreamFormError(null);
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
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {channels.map((channel) => (
              <tr key={channel.name} className="hover:bg-gray-50 cursor-pointer">
                <td
                  className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900"
                  onClick={() => handleChannelClick(channel.name)}
                >
                  {channel.name}
                </td>
                <td
                  className="px-6 py-4 whitespace-nowrap text-sm text-gray-500"
                  onClick={() => handleChannelClick(channel.name)}
                >
                  {getStreamCount(channel.name)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteChannel(channel.name);
                    }}
                    className="text-red-600 hover:text-red-900 hover:bg-red-50"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
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

      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent className="w-[400px] sm:w-[540px] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>{selectedChannel}</SheetTitle>
            <SheetDescription>
              Manage streams for this channel
            </SheetDescription>
          </SheetHeader>
          <div className="mt-6">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-sm font-medium">Streams</h3>
              <Button
                size="sm"
                variant="outline"
                onClick={() => setStreamDialogOpen(true)}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add stream
              </Button>
            </div>
            {selectedChannel && getChannelStreams(selectedChannel).length === 0 ? (
              <p className="text-sm text-gray-500">No streams found for this channel.</p>
            ) : (
              <div className="space-y-2">
                {selectedChannel &&
                  getChannelStreams(selectedChannel).map((stream) => (
                    <div
                      key={stream.info_hash}
                      className="flex items-center justify-between p-3 border rounded-lg hover:bg-gray-50"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-900 truncate">
                          {stream.info_hash}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDeleteStream(stream.info_hash)}
                        className="ml-2 text-red-600 hover:text-red-900 hover:bg-red-50"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <Dialog open={streamDialogOpen} onOpenChange={handleStreamDialogOpenChange}>
        <DialogContent>
          <form onSubmit={handleCreateStream}>
            <DialogHeader>
              <DialogTitle>Add stream</DialogTitle>
              <DialogDescription>
                Add a new stream to {selectedChannel}
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="stream-info-hash">Info Hash</Label>
                <Input
                  id="stream-info-hash"
                  value={streamInfoHash}
                  onChange={(e) => setStreamInfoHash(e.target.value)}
                  placeholder="Enter stream info hash"
                  disabled={streamSubmitting}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="stream-channel">Channel</Label>
                <Input
                  id="stream-channel"
                  value={selectedChannel || ""}
                  disabled
                  readOnly
                />
              </div>
              {streamFormError && (
                <p className="text-sm text-red-600">{streamFormError}</p>
              )}
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setStreamDialogOpen(false)}
                disabled={streamSubmitting}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={streamSubmitting}>
                {streamSubmitting ? "Creating..." : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
