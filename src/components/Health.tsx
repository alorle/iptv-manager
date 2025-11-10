import { CheckCircle2, Loader2, XCircle } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { $api } from "@/lib/api/client";
import { toTitleCase } from "@/lib/utils";

function HealthCheck() {
  const { data, isLoading, isError, dataUpdatedAt } = $api.useQuery(
    "get",
    "/health",
    {},
    { refetchInterval: 5000 }
  );

  const formatTime = (timestamp: number) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  let content;

  if (isLoading) {
    content = (
      <div className="flex flex-col items-center gap-4 py-8">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        <p className="text-sm text-muted-foreground">Checking system health...</p>
      </div>
    );
  } else if (isError) {
    content = (
      <div className="flex flex-col items-center gap-4 py-8">
        <XCircle className="h-12 w-12 text-destructive" />
        <Badge variant="destructive">Unhealthy</Badge>
        <p className="text-sm text-muted-foreground">Failed to connect to API</p>
      </div>
    );
  } else if (data) {
    content = (
      <div className="space-y-6">
        <div className="flex flex-col items-center gap-3">
          <CheckCircle2 className="h-12 w-12 text-green-600" />
          <Badge variant="success">{toTitleCase(data.status)}</Badge>
        </div>

        <div className="space-y-3">
          <div className="flex justify-between rounded-lg border bg-muted/50 p-3">
            <span className="text-sm font-medium">Version</span>
            <span className="font-mono text-sm">{data.version}</span>
          </div>

          <div className="flex justify-between rounded-lg border bg-muted/50 p-3">
            <span className="text-sm font-medium">Server Time</span>
            <span className="font-mono text-sm">{new Date(data.timestamp).toLocaleString()}</span>
          </div>

          <div className="flex justify-between rounded-lg border bg-muted/50 p-3">
            <span className="text-sm font-medium">Last Check</span>
            <span className="font-mono text-sm">{formatTime(dataUpdatedAt)}</span>
          </div>
        </div>

        <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
          <div className="h-2 w-2 rounded-full bg-green-600 animate-pulse" />
          <span>Live</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">IPTV Manager</CardTitle>
          <CardDescription>System Health Status</CardDescription>
        </CardHeader>

        <CardContent className="space-y-6">{content}</CardContent>
      </Card>
    </div>
  );
}

export default HealthCheck;
