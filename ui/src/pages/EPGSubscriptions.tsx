import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { ChevronDown, ChevronUp, Search, AlertCircle } from "lucide-react";

interface EPGChannel {
  id: string;
  name: string;
  logo: string;
  category: string;
  language: string;
  epg_id: string;
}

interface Subscription {
  epg_channel_id: string;
  enabled: boolean;
  manual_override: boolean;
}

interface CategoryGroup {
  category: string;
  channels: EPGChannel[];
}

export default function EPGSubscriptions() {
  const [channels, setChannels] = useState<EPGChannel[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const [openCategories, setOpenCategories] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadChannels();
    loadSubscriptions();
  }, []);

  const loadChannels = async () => {
    try {
      const response = await fetch("/api/epg/channels");
      if (!response.ok) throw new Error("Failed to load channels");
      const data = await response.json();
      setChannels(data || []);
    } catch (error) {
      toast.error("Failed to load EPG channels");
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const loadSubscriptions = async () => {
    try {
      const response = await fetch("/api/subscriptions");
      if (!response.ok) throw new Error("Failed to load subscriptions");
      const data = await response.json();
      setSubscriptions(data || []);
    } catch (error) {
      console.error("Failed to load subscriptions", error);
    }
  };

  const handleImport = async () => {
    setImporting(true);
    try {
      const response = await fetch("/api/epg/import", { method: "POST" });
      if (!response.ok) throw new Error("Failed to import EPG data");
      toast.success("EPG import started successfully");
      // Reload channels after import
      await loadChannels();
    } catch (error) {
      toast.error("Failed to import EPG data");
      console.error(error);
    } finally {
      setImporting(false);
    }
  };

  const isSubscribed = (channelId: string): boolean => {
    return subscriptions.some((sub) => sub.epg_channel_id === channelId && sub.enabled);
  };

  const toggleSubscription = async (channel: EPGChannel) => {
    const subscribed = isSubscribed(channel.id);

    try {
      if (subscribed) {
        // Unsubscribe
        const response = await fetch(`/api/subscriptions/${channel.id}`, {
          method: "DELETE",
        });
        if (!response.ok) throw new Error("Failed to unsubscribe");
        setSubscriptions((prev) => prev.filter((sub) => sub.epg_channel_id !== channel.id));
        toast.success(`Unsubscribed from ${channel.name}`);
      } else {
        // Subscribe
        const response = await fetch("/api/subscriptions", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ epg_channel_id: channel.id }),
        });
        if (!response.ok) throw new Error("Failed to subscribe");
        const newSub = await response.json();
        setSubscriptions((prev) => [...prev, newSub]);
        toast.success(`Subscribed to ${channel.name}`);
      }
    } catch (error) {
      toast.error(subscribed ? "Failed to unsubscribe" : "Failed to subscribe");
      console.error(error);
    }
  };

  const toggleCategory = (category: string) => {
    setOpenCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  const filteredChannels = channels.filter((channel) => {
    const matchesSearch = searchTerm
      ? channel.name.toLowerCase().includes(searchTerm.toLowerCase())
      : true;
    const matchesCategory = selectedCategory
      ? channel.category.toLowerCase() === selectedCategory.toLowerCase()
      : true;
    return matchesSearch && matchesCategory;
  });

  const groupedChannels: CategoryGroup[] = filteredChannels.reduce(
    (acc: CategoryGroup[], channel) => {
      const existingGroup = acc.find(
        (g) => g.category.toLowerCase() === channel.category.toLowerCase()
      );
      if (existingGroup) {
        existingGroup.channels.push(channel);
      } else {
        acc.push({ category: channel.category, channels: [channel] });
      }
      return acc;
    },
    []
  );

  // Sort groups by category name
  groupedChannels.sort((a, b) => a.category.localeCompare(b.category));

  const categories = Array.from(new Set(channels.map((ch) => ch.category))).sort();

  if (loading) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">EPG Channel Subscriptions</h1>
        <p className="text-gray-600">Loading channels...</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-2">EPG Channel Subscriptions</h1>
        <p className="text-gray-600">
          Browse and subscribe to EPG channels to curate your channel list.
        </p>
      </div>

      <div className="mb-6 flex gap-4 items-center">
        <Button onClick={handleImport} disabled={importing}>
          {importing ? "Importing..." : "Setup Channels from EPG"}
        </Button>
        <Badge variant="secondary">
          {filteredChannels.length} channel{filteredChannels.length !== 1 ? "s" : ""}
        </Badge>
        <Badge>
          {subscriptions.filter((s) => s.enabled).length} subscribed
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
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className="border rounded-md px-3 py-2 bg-white"
        >
          <option value="">All Categories</option>
          {categories.map((cat) => (
            <option key={cat} value={cat}>
              {cat}
            </option>
          ))}
        </select>
      </div>

      {filteredChannels.length === 0 && (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center gap-2 text-gray-600">
              <AlertCircle className="h-5 w-5" />
              <p>
                {channels.length === 0
                  ? "No EPG channels available. Click 'Setup Channels from EPG' to import."
                  : "No channels match your search criteria."}
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-4">
        {groupedChannels.map((group) => {
          const isOpen = openCategories.has(group.category);
          const subscribedCount = group.channels.filter((ch) => isSubscribed(ch.id)).length;

          return (
            <Collapsible
              key={group.category}
              open={isOpen}
              onOpenChange={() => toggleCategory(group.category)}
            >
              <Card>
                <CollapsibleTrigger className="w-full">
                  <CardHeader className="flex flex-row items-center justify-between py-4 cursor-pointer hover:bg-gray-50">
                    <div className="flex items-center gap-3">
                      {isOpen ? (
                        <ChevronUp className="h-5 w-5 text-gray-500" />
                      ) : (
                        <ChevronDown className="h-5 w-5 text-gray-500" />
                      )}
                      <CardTitle className="text-lg">{group.category}</CardTitle>
                      <Badge variant="secondary">
                        {group.channels.length}
                      </Badge>
                      {subscribedCount > 0 && (
                        <Badge>
                          {subscribedCount} subscribed
                        </Badge>
                      )}
                    </div>
                  </CardHeader>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <CardContent className="pt-0">
                    <div className="space-y-3">
                      {group.channels.map((channel) => {
                        const subscribed = isSubscribed(channel.id);
                        return (
                          <div
                            key={channel.id}
                            className="flex items-center justify-between p-3 border rounded-lg hover:bg-gray-50"
                          >
                            <div className="flex items-center gap-3 flex-1">
                              {channel.logo ? (
                                <img
                                  src={channel.logo}
                                  alt={channel.name}
                                  className="h-10 w-10 object-contain rounded"
                                />
                              ) : (
                                <div className="h-10 w-10 bg-gray-200 rounded flex items-center justify-center text-gray-500 text-xs">
                                  No Logo
                                </div>
                              )}
                              <div className="flex-1">
                                <h3 className="font-medium">{channel.name}</h3>
                                <div className="flex gap-2 mt-1">
                                  {channel.language && (
                                    <Badge variant="outline" className="text-xs">
                                      {channel.language}
                                    </Badge>
                                  )}
                                  <span className="text-xs text-gray-500">
                                    ID: {channel.epg_id}
                                  </span>
                                </div>
                              </div>
                            </div>
                            <Button
                              variant={subscribed ? "secondary" : "default"}
                              onClick={() => toggleSubscription(channel)}
                            >
                              {subscribed ? "Unsubscribe" : "Subscribe"}
                            </Button>
                          </div>
                        );
                      })}
                    </div>
                  </CardContent>
                </CollapsibleContent>
              </Card>
            </Collapsible>
          );
        })}
      </div>
    </div>
  );
}
