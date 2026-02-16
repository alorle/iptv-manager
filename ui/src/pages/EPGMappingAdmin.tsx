import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Search, AlertCircle, CheckCircle, Edit2, X } from "lucide-react";

interface EPGMapping {
  epg_id: string;
  source: "auto" | "manual";
  last_synced: string;
}

interface Channel {
  name: string;
  status: string;
  epg_mapping?: EPGMapping;
}

interface StreamHash {
  info_hash: string;
  channel_name: string;
}

interface EPGChannel {
  id: string;
  name: string;
  logo: string;
  category: string;
  language: string;
  epg_id: string;
}

interface ChannelWithMapping extends Channel {
  mappingStatus: "auto-matched" | "manually-mapped" | "unmapped";
  streams: StreamHash[];
}

export default function EPGMappingAdmin() {
  const [channels, setChannels] = useState<ChannelWithMapping[]>([]);
  const [epgChannels, setEPGChannels] = useState<EPGChannel[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [editingChannel, setEditingChannel] = useState<ChannelWithMapping | null>(null);
  const [selectedEPGID, setSelectedEPGID] = useState("");
  const [epgSearchTerm, setEPGSearchTerm] = useState("");

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      await Promise.all([loadChannels(), loadEPGChannels()]);
    } catch (error) {
      toast.error("Failed to load data");
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const loadChannels = async () => {
    try {
      // Load channels and streams in parallel
      const [channelsRes, streamsRes] = await Promise.all([
        fetch("/api/channels"),
        fetch("/api/streams"),
      ]);

      if (!channelsRes.ok || !streamsRes.ok) {
        throw new Error("Failed to load channels or streams");
      }

      const channelsData: Channel[] = await channelsRes.json();
      const streamsData: StreamHash[] = await streamsRes.json();

      // Build mapping of channel name to streams
      const streamsByChannel = new Map<string, StreamHash[]>();
      for (const stream of streamsData) {
        const existing = streamsByChannel.get(stream.channel_name) || [];
        existing.push(stream);
        streamsByChannel.set(stream.channel_name, existing);
      }

      // Enhance channels with mapping status and streams
      const enhanced: ChannelWithMapping[] = channelsData.map((ch) => {
        let mappingStatus: "auto-matched" | "manually-mapped" | "unmapped";
        if (!ch.epg_mapping) {
          mappingStatus = "unmapped";
        } else if (ch.epg_mapping.source === "manual") {
          mappingStatus = "manually-mapped";
        } else {
          mappingStatus = "auto-matched";
        }

        return {
          ...ch,
          mappingStatus,
          streams: streamsByChannel.get(ch.name) || [],
        };
      });

      setChannels(enhanced);
    } catch (error) {
      toast.error("Failed to load channels");
      console.error(error);
    }
  };

  const loadEPGChannels = async () => {
    try {
      const response = await fetch("/api/epg/channels");
      if (!response.ok) throw new Error("Failed to load EPG channels");
      const data = await response.json();
      setEPGChannels(data || []);
    } catch (error) {
      toast.error("Failed to load EPG channels");
      console.error(error);
    }
  };

  const handleUpdateMapping = async () => {
    if (!editingChannel || !selectedEPGID) {
      toast.error("Please select an EPG channel");
      return;
    }

    try {
      const response = await fetch(`/api/epg/mappings/${encodeURIComponent(editingChannel.name)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ epg_id: selectedEPGID }),
      });

      if (!response.ok) throw new Error("Failed to update mapping");

      toast.success(`Updated EPG mapping for ${editingChannel.name}`);
      setEditingChannel(null);
      setSelectedEPGID("");
      setEPGSearchTerm("");
      await loadChannels();
    } catch (error) {
      toast.error("Failed to update mapping");
      console.error(error);
    }
  };

  const handleClearMapping = async (channelName: string) => {
    try {
      const response = await fetch(`/api/epg/mappings/${encodeURIComponent(channelName)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ epg_id: "" }),
      });

      if (!response.ok) throw new Error("Failed to clear mapping");

      toast.success(`Cleared EPG mapping for ${channelName}`);
      await loadChannels();
    } catch (error) {
      toast.error("Failed to clear mapping");
      console.error(error);
    }
  };

  const filteredChannels = channels.filter((channel) => {
    const matchesSearch = searchTerm
      ? channel.name.toLowerCase().includes(searchTerm.toLowerCase())
      : true;
    const matchesStatus = filterStatus
      ? channel.mappingStatus === filterStatus
      : true;
    return matchesSearch && matchesStatus;
  });

  const filteredEPGChannels = epgChannels.filter((ch) =>
    epgSearchTerm
      ? ch.name.toLowerCase().includes(epgSearchTerm.toLowerCase()) ||
        ch.epg_id.toLowerCase().includes(epgSearchTerm.toLowerCase())
      : true
  );

  const getMappingBadge = (status: "auto-matched" | "manually-mapped" | "unmapped") => {
    switch (status) {
      case "auto-matched":
        return <Badge variant="secondary">Auto-matched</Badge>;
      case "manually-mapped":
        return <Badge>Manual</Badge>;
      case "unmapped":
        return <Badge variant="outline">Unmapped</Badge>;
    }
  };

  if (loading) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">EPG Mapping Admin</h1>
        <p className="text-gray-600">Loading channels...</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-2">EPG Mapping Admin</h1>
        <p className="text-gray-600">
          Manually map EPG channels to Acestream streams when automatic matching fails.
        </p>
      </div>

      <div className="mb-6 flex gap-4 items-center">
        <Badge variant="secondary">
          {filteredChannels.length} channel{filteredChannels.length !== 1 ? "s" : ""}
        </Badge>
        <Badge>
          {channels.filter((c) => c.mappingStatus === "auto-matched").length} auto-matched
        </Badge>
        <Badge>
          {channels.filter((c) => c.mappingStatus === "manually-mapped").length} manual
        </Badge>
        <Badge variant="outline">
          {channels.filter((c) => c.mappingStatus === "unmapped").length} unmapped
        </Badge>
      </div>

      <div className="mb-6 flex gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
          <Input
            type="text"
            placeholder="Search channels by name..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-10"
          />
        </div>
        <select
          value={filterStatus}
          onChange={(e) => setFilterStatus(e.target.value)}
          className="border rounded-md px-3 py-2 bg-white"
        >
          <option value="">All Statuses</option>
          <option value="unmapped">Unmapped</option>
          <option value="auto-matched">Auto-matched</option>
          <option value="manually-mapped">Manually Mapped</option>
        </select>
      </div>

      {filteredChannels.length === 0 && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-2 text-gray-600">
              <AlertCircle className="h-5 w-5" />
              <p>No channels match your search criteria.</p>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {filteredChannels.map((channel) => (
          <Card key={channel.name}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3 flex-1">
                  <CardTitle className="text-lg">{channel.name}</CardTitle>
                  {getMappingBadge(channel.mappingStatus)}
                  {channel.streams.length > 0 && (
                    <Badge variant="secondary">{channel.streams.length} stream(s)</Badge>
                  )}
                </div>
                <div className="flex gap-2">
                  <Dialog
                    open={editingChannel?.name === channel.name}
                    onOpenChange={(open) => {
                      if (!open) {
                        setEditingChannel(null);
                        setSelectedEPGID("");
                        setEPGSearchTerm("");
                      }
                    }}
                  >
                    <DialogTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          setEditingChannel(channel);
                          setSelectedEPGID(channel.epg_mapping?.epg_id || "");
                        }}
                      >
                        <Edit2 className="h-4 w-4 mr-1" />
                        Edit Mapping
                      </Button>
                    </DialogTrigger>
                    <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
                      <DialogHeader>
                        <DialogTitle>Edit EPG Mapping for {channel.name}</DialogTitle>
                        <DialogDescription>
                          Select an EPG channel to map to this stream channel.
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4 mt-4">
                        <div>
                          <Label>Search EPG Channels</Label>
                          <Input
                            type="text"
                            placeholder="Search by name or EPG ID..."
                            value={epgSearchTerm}
                            onChange={(e) => setEPGSearchTerm(e.target.value)}
                            className="mt-1"
                          />
                        </div>
                        <div className="border rounded-lg max-h-96 overflow-y-auto">
                          {filteredEPGChannels.length === 0 ? (
                            <div className="p-4 text-center text-gray-500">
                              No EPG channels found
                            </div>
                          ) : (
                            <div className="divide-y">
                              {filteredEPGChannels.map((epgCh) => (
                                <div
                                  key={epgCh.id}
                                  className={`p-3 cursor-pointer hover:bg-gray-50 ${
                                    selectedEPGID === epgCh.epg_id ? "bg-blue-50" : ""
                                  }`}
                                  onClick={() => setSelectedEPGID(epgCh.epg_id)}
                                >
                                  <div className="flex items-center gap-3">
                                    {epgCh.logo ? (
                                      <img
                                        src={epgCh.logo}
                                        alt={epgCh.name}
                                        className="h-8 w-8 object-contain rounded"
                                      />
                                    ) : (
                                      <div className="h-8 w-8 bg-gray-200 rounded" />
                                    )}
                                    <div className="flex-1">
                                      <div className="font-medium">{epgCh.name}</div>
                                      <div className="text-sm text-gray-500">
                                        ID: {epgCh.epg_id} | {epgCh.category}
                                      </div>
                                    </div>
                                    {selectedEPGID === epgCh.epg_id && (
                                      <CheckCircle className="h-5 w-5 text-blue-600" />
                                    )}
                                  </div>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            onClick={() => {
                              setEditingChannel(null);
                              setSelectedEPGID("");
                              setEPGSearchTerm("");
                            }}
                          >
                            Cancel
                          </Button>
                          <Button
                            onClick={handleUpdateMapping}
                            disabled={!selectedEPGID}
                          >
                            Save Mapping
                          </Button>
                        </div>
                      </div>
                    </DialogContent>
                  </Dialog>
                  {channel.epg_mapping && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleClearMapping(channel.name)}
                    >
                      <X className="h-4 w-4 mr-1" />
                      Clear
                    </Button>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="pt-0">
              <div className="space-y-2 text-sm">
                {channel.epg_mapping ? (
                  <div className="flex gap-4">
                    <div>
                      <span className="text-gray-600">EPG ID:</span>{" "}
                      <span className="font-mono">{channel.epg_mapping.epg_id}</span>
                    </div>
                    <div>
                      <span className="text-gray-600">Last Synced:</span>{" "}
                      {new Date(channel.epg_mapping.last_synced).toLocaleString()}
                    </div>
                  </div>
                ) : (
                  <div className="text-gray-500">
                    No EPG mapping set. Click "Edit Mapping" to add one.
                  </div>
                )}
                {channel.streams.length > 0 && (
                  <div className="mt-3 pt-3 border-t">
                    <div className="text-gray-600 mb-2">Acestream Hashes:</div>
                    <div className="space-y-1">
                      {channel.streams.map((stream) => (
                        <div key={stream.info_hash} className="font-mono text-xs bg-gray-100 p-2 rounded">
                          {stream.info_hash}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
