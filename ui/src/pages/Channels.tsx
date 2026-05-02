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
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { Trash2, Plus, Activity, Loader2, AlertTriangle } from "lucide-react";

interface Stream {
  info_hash: string;
  channel_name: string;
}

interface ProbeResult {
  info_hash: string;
  timestamp: string;
  available: boolean;
  startup_latency_ms: number;
  peer_count: number;
  download_speed: number;
  status: string;
  error_message?: string;
}

interface ChannelHealth {
  name: string;
  status: string;
  stream_count: number;
  best_score: number;
  health_level: "green" | "yellow" | "red" | "unknown";
  last_probe?: ProbeResult;
  watching: number;
}

interface SystemHealth {
  status: string;
  db: string;
  acestream_engine: string;
}

interface DashboardResponse {
  system: SystemHealth;
  channels: ChannelHealth[];
  sessions: { info_hash: string; client_count: number }[];
}

function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const then = new Date(timestamp).getTime();
  const diffMs = now - then;
  const diffMin = Math.floor(diffMs / 60000);

  if (diffMin < 1) return "just now";
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHrs = Math.floor(diffMin / 60);
  if (diffHrs < 24) return `${diffHrs}h ago`;
  return `${Math.floor(diffHrs / 24)}d ago`;
}

const healthDotColor: Record<string, string> = {
  green: "bg-green-500",
  yellow: "bg-yellow-500",
  red: "bg-red-500",
  unknown: "bg-gray-300",
};

export default function Channels() {
  const [dashboard, setDashboard] = useState<DashboardResponse | null>(null);
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
  const [probingStream, setProbingStream] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [dashboardRes, streamsRes] = await Promise.all([
        fetch("/api/dashboard"),
        fetch("/api/streams"),
      ]);

      if (!dashboardRes.ok) {
        throw new Error(`Failed to fetch dashboard: ${dashboardRes.status}`);
      }
      if (!streamsRes.ok) {
        throw new Error(`Failed to fetch streams: ${streamsRes.status}`);
      }

      const dashboardData = await dashboardRes.json();
      const streamsData = await streamsRes.json();

      setDashboard(dashboardData);
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
        headers: { "Content-Type": "application/json" },
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
    if (
      !confirm(
        `Are you sure you want to delete channel "${name}"? This will also delete all associated streams.`
      )
    ) {
      return;
    }

    try {
      const response = await fetch(
        `/api/channels/${encodeURIComponent(name)}`,
        { method: "DELETE" }
      );

      if (!response.ok) {
        const errorData = await response
          .json()
          .catch(() => ({ error: "Failed to delete channel" }));
        toast.error(errorData.error || "Failed to delete channel");
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
      const response = await fetch(
        `/api/streams/${encodeURIComponent(infoHash)}`,
        { method: "DELETE" }
      );

      if (!response.ok) {
        const errorData = await response
          .json()
          .catch(() => ({ error: "Failed to delete stream" }));
        toast.error(errorData.error || "Failed to delete stream");
        return;
      }

      toast.success(`Stream deleted successfully`);
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
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          info_hash: streamInfoHash,
          channel_name: selectedChannel,
        }),
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
      setStreamFormError(
        err instanceof Error ? err.message : "An error occurred"
      );
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

  const handleTestStream = async (infoHash: string) => {
    setProbingStream(infoHash);
    try {
      const response = await fetch(
        `/api/probes/${encodeURIComponent(infoHash)}`,
        { method: "POST" }
      );

      if (!response.ok) {
        const errorData = await response
          .json()
          .catch(() => ({ error: "Probe failed" }));
        toast.error(errorData.error || "Probe failed");
        return;
      }

      const result: ProbeResult = await response.json();
      if (result.available) {
        toast.success(
          `Stream OK — ${result.peer_count} peers, ${(result.download_speed / 1024).toFixed(0)} KB/s, ${result.startup_latency_ms}ms startup`
        );
      } else {
        toast.error(
          `Stream unavailable — ${result.error_message || "unknown error"}`
        );
      }

      await fetchData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Probe failed");
    } finally {
      setProbingStream(null);
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

  const channels = dashboard?.channels ?? [];

  const renderSystemBanner = () => {
    if (!dashboard || dashboard.system.status === "ok") return null;

    return (
      <div className="mb-4 flex items-center gap-2 rounded-lg border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800">
        <AlertTriangle className="h-4 w-4 flex-shrink-0" />
        <span>
          System degraded —{" "}
          {dashboard.system.db !== "ok" && `DB: ${dashboard.system.db}`}
          {dashboard.system.db !== "ok" &&
            dashboard.system.acestream_engine !== "ok" &&
            " · "}
          {dashboard.system.acestream_engine !== "ok" &&
            `Engine: ${dashboard.system.acestream_engine}`}
        </span>
      </div>
    );
  };

  const renderContent = () => {
    if (channels.length === 0) {
      return (
        <p className="text-gray-600">
          No channels found. Create your first channel to get started.
        </p>
      );
    }

    return (
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200 border border-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-10">
                Health
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Streams
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Last Probe
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Score
              </th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {channels.map((ch) => (
              <tr key={ch.name} className="hover:bg-gray-50 cursor-pointer">
                <td
                  className="px-4 py-4"
                  onClick={() => handleChannelClick(ch.name)}
                >
                  <span
                    className={`inline-block h-3 w-3 rounded-full ${healthDotColor[ch.health_level]}`}
                    title={ch.health_level}
                  />
                </td>
                <td
                  className="px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900"
                  onClick={() => handleChannelClick(ch.name)}
                >
                  <div className="flex items-center gap-2">
                    {ch.name}
                    {ch.watching > 0 && (
                      <Badge variant="secondary" className="text-xs">
                        {ch.watching} watching
                      </Badge>
                    )}
                    {ch.status === "archived" && (
                      <Badge variant="outline" className="text-xs text-gray-400">
                        archived
                      </Badge>
                    )}
                  </div>
                </td>
                <td
                  className="px-4 py-4 whitespace-nowrap text-sm text-gray-500"
                  onClick={() => handleChannelClick(ch.name)}
                >
                  {ch.stream_count}
                </td>
                <td
                  className="px-4 py-4 whitespace-nowrap text-sm text-gray-500"
                  onClick={() => handleChannelClick(ch.name)}
                >
                  {ch.last_probe ? (
                    <span
                      className={
                        ch.last_probe.available
                          ? "text-green-700"
                          : "text-red-600"
                      }
                    >
                      {ch.last_probe.available ? "OK" : "Failed"} ·{" "}
                      {formatRelativeTime(ch.last_probe.timestamp)}
                    </span>
                  ) : (
                    <span className="text-gray-400">No probes</span>
                  )}
                </td>
                <td
                  className="px-4 py-4 whitespace-nowrap text-sm text-gray-500"
                  onClick={() => handleChannelClick(ch.name)}
                >
                  {ch.best_score > 0
                    ? `${(ch.best_score * 100).toFixed(0)}%`
                    : "—"}
                </td>
                <td className="px-4 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDeleteChannel(ch.name);
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

  const selectedChannelHealth = channels.find(
    (ch) => ch.name === selectedChannel
  );

  return (
    <div>
      {renderSystemBanner()}

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
            <SheetTitle>
              <div className="flex items-center gap-2">
                {selectedChannelHealth && (
                  <span
                    className={`inline-block h-3 w-3 rounded-full ${healthDotColor[selectedChannelHealth.health_level]}`}
                  />
                )}
                {selectedChannel}
              </div>
            </SheetTitle>
            <SheetDescription>
              Manage streams for this channel
            </SheetDescription>
          </SheetHeader>

          {selectedChannelHealth?.last_probe && (
            <div className="mt-4 rounded-lg border bg-gray-50 p-3 text-sm">
              <p className="font-medium mb-1">Last probe</p>
              <p className="text-gray-600">
                {selectedChannelHealth.last_probe.available ? "✓ Available" : "✗ Unavailable"}{" · "}
                {formatRelativeTime(selectedChannelHealth.last_probe.timestamp)}
              </p>
              {selectedChannelHealth.last_probe.available && (
                <p className="text-gray-500 text-xs mt-1">
                  {selectedChannelHealth.last_probe.peer_count} peers ·{" "}
                  {(selectedChannelHealth.last_probe.download_speed / 1024).toFixed(0)} KB/s ·{" "}
                  {selectedChannelHealth.last_probe.startup_latency_ms}ms startup
                </p>
              )}
              {selectedChannelHealth.best_score > 0 && (
                <p className="text-gray-500 text-xs mt-1">
                  Quality score: {(selectedChannelHealth.best_score * 100).toFixed(0)}%
                </p>
              )}
            </div>
          )}

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
            {selectedChannel &&
            getChannelStreams(selectedChannel).length === 0 ? (
              <p className="text-sm text-gray-500">
                No streams found for this channel.
              </p>
            ) : (
              <div className="space-y-2">
                {selectedChannel &&
                  getChannelStreams(selectedChannel).map((stream) => (
                    <div
                      key={stream.info_hash}
                      className="flex items-center justify-between p-3 border rounded-lg hover:bg-gray-50"
                    >
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-900 truncate font-mono">
                          {stream.info_hash}
                        </p>
                      </div>
                      <div className="flex items-center gap-1 ml-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleTestStream(stream.info_hash)}
                          disabled={probingStream === stream.info_hash}
                          className="text-blue-600 hover:text-blue-900 hover:bg-blue-50"
                          title="Test stream"
                        >
                          {probingStream === stream.info_hash ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <Activity className="h-4 w-4" />
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDeleteStream(stream.info_hash)}
                          className="text-red-600 hover:text-red-900 hover:bg-red-50"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
              </div>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <Dialog
        open={streamDialogOpen}
        onOpenChange={handleStreamDialogOpenChange}
      >
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
