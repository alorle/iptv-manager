import { useEffect, useState, useRef, useCallback } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { RefreshCw, Pause, Play, Activity, AlertTriangle, CheckCircle2, XCircle } from "lucide-react";

interface StreamCounters {
  streams_started: number;
  stream_start_failures: number;
  streams_stopped: number;
  stream_stop_failures: number;
  reconnection_attempts: number;
  reconnection_successes: number;
  clients_served: number;
}

interface SessionDiagnostic {
  info_hash: string;
  state: string;
  stream_url?: string;
  engine_pid?: string;
  clients: string[];
  client_count: number;
  error?: string;
  created_at: string;
}

interface StreamDiagnostics {
  uptime: number; // nanoseconds
  counters: StreamCounters;
  sessions: SessionDiagnostic[];
  engine_healthy: boolean;
}

function formatUptime(nanos: number): string {
  const totalSeconds = Math.floor(nanos / 1e9);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

function formatSessionAge(createdAt: string): string {
  const diffMs = Date.now() - new Date(createdAt).getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ${diffSec % 60}s`;
  const diffHrs = Math.floor(diffMin / 60);
  return `${diffHrs}h ${diffMin % 60}m`;
}

const stateColors: Record<string, string> = {
  streaming: "bg-green-100 text-green-800",
  starting: "bg-yellow-100 text-yellow-800",
  error: "bg-red-100 text-red-800",
};

export default function DebugStreams() {
  const [data, setData] = useState<StreamDiagnostics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setError(null);
      const response = await fetch("/api/debug/streams");
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setLastRefresh(new Date());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchData, 5000);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, fetchData]);

  if (loading && !data) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Debug — Streams</h1>
        <p className="text-gray-600">Loading diagnostics...</p>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div>
        <h1 className="text-2xl font-bold mb-4">Debug — Streams</h1>
        <p className="text-red-600">Error: {error}</p>
      </div>
    );
  }

  if (!data) return null;

  const { counters } = data;
  const leakedSessions =
    counters.streams_started - counters.streams_stopped - counters.stream_stop_failures;

  return (
    <div>
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">Debug — Streams</h1>
          <p className="text-sm text-gray-500 mt-1">
            Uptime: {formatUptime(data.uptime)}
            {lastRefresh && (
              <span className="ml-3">
                Last refresh: {lastRefresh.toLocaleTimeString()}
              </span>
            )}
            {error && (
              <span className="ml-3 text-red-500">Refresh failed</span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setAutoRefresh(!autoRefresh)}
            title={autoRefresh ? "Pause auto-refresh" : "Resume auto-refresh"}
          >
            {autoRefresh ? (
              <Pause className="h-4 w-4" />
            ) : (
              <Play className="h-4 w-4" />
            )}
          </Button>
          <Button variant="outline" size="sm" onClick={fetchData}>
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Engine Health Banner */}
      <div
        className={`mb-6 flex items-center gap-2 rounded-lg border p-3 text-sm ${
          data.engine_healthy
            ? "border-green-200 bg-green-50 text-green-800"
            : "border-red-200 bg-red-50 text-red-800"
        }`}
      >
        {data.engine_healthy ? (
          <CheckCircle2 className="h-4 w-4 flex-shrink-0" />
        ) : (
          <XCircle className="h-4 w-4 flex-shrink-0" />
        )}
        <span>
          AceStream Engine:{" "}
          {data.engine_healthy ? "Healthy" : "Unreachable"}
        </span>
      </div>

      {/* Counters Grid */}
      <div className="mb-6">
        <h2 className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-3">
          Lifecycle Counters
        </h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <CounterCard
            label="Started"
            value={counters.streams_started}
            variant="default"
          />
          <CounterCard
            label="Stopped"
            value={counters.streams_stopped}
            variant="default"
          />
          <CounterCard
            label="Start Failures"
            value={counters.stream_start_failures}
            variant={counters.stream_start_failures > 0 ? "warning" : "default"}
          />
          <CounterCard
            label="Stop Failures"
            value={counters.stream_stop_failures}
            variant={counters.stream_stop_failures > 0 ? "warning" : "default"}
          />
          <CounterCard
            label="Reconnection Attempts"
            value={counters.reconnection_attempts}
            variant={counters.reconnection_attempts > 0 ? "warning" : "default"}
          />
          <CounterCard
            label="Reconnection OK"
            value={counters.reconnection_successes}
            variant="default"
          />
          <CounterCard
            label="Clients Served"
            value={counters.clients_served}
            variant="default"
          />
          <CounterCard
            label="Leaked Sessions"
            value={leakedSessions}
            variant={leakedSessions > 0 ? "danger" : "default"}
          />
        </div>
      </div>

      {/* Active Sessions */}
      <div>
        <h2 className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-3">
          Active Sessions ({data.sessions.length})
        </h2>
        {data.sessions.length === 0 ? (
          <div className="rounded-lg border border-dashed border-gray-300 p-8 text-center text-sm text-gray-500">
            <Activity className="h-8 w-8 mx-auto mb-2 text-gray-300" />
            No active sessions
          </div>
        ) : (
          <div className="space-y-3">
            {data.sessions.map((session) => (
              <SessionCard key={session.info_hash} session={session} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function CounterCard({
  label,
  value,
  variant,
}: {
  label: string;
  value: number;
  variant: "default" | "warning" | "danger";
}) {
  const borderColor =
    variant === "danger"
      ? "border-red-200 bg-red-50"
      : variant === "warning"
        ? "border-yellow-200 bg-yellow-50"
        : "border-gray-200 bg-white";

  const valueColor =
    variant === "danger"
      ? "text-red-700"
      : variant === "warning"
        ? "text-yellow-700"
        : "text-gray-900";

  return (
    <div className={`rounded-lg border p-3 ${borderColor}`}>
      <p className="text-xs text-gray-500">{label}</p>
      <p className={`text-2xl font-bold ${valueColor}`}>{value}</p>
    </div>
  );
}

function SessionCard({ session }: { session: SessionDiagnostic }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center gap-2">
          <Badge
            className={stateColors[session.state] ?? "bg-gray-100 text-gray-800"}
          >
            {session.state}
          </Badge>
          <span className="text-sm text-gray-500">
            {formatSessionAge(session.created_at)} active
          </span>
        </div>
        <Badge variant="secondary">{session.client_count} clients</Badge>
      </div>

      <div className="space-y-1 text-sm">
        <div className="flex gap-2">
          <span className="text-gray-500 w-20 flex-shrink-0">Infohash</span>
          <span className="font-mono text-gray-900 truncate">
            {session.info_hash}
          </span>
        </div>
        {session.engine_pid && (
          <div className="flex gap-2">
            <span className="text-gray-500 w-20 flex-shrink-0">Engine PID</span>
            <span className="font-mono text-gray-700">{session.engine_pid}</span>
          </div>
        )}
        {session.stream_url && (
          <div className="flex gap-2">
            <span className="text-gray-500 w-20 flex-shrink-0">Stream URL</span>
            <span className="font-mono text-gray-700 truncate text-xs">
              {session.stream_url}
            </span>
          </div>
        )}
        {session.clients.length > 0 && (
          <div className="flex gap-2">
            <span className="text-gray-500 w-20 flex-shrink-0">Clients</span>
            <div className="flex flex-wrap gap-1">
              {session.clients.map((pid) => (
                <span
                  key={pid}
                  className="inline-block rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-600"
                >
                  {pid}
                </span>
              ))}
            </div>
          </div>
        )}
        {session.error && (
          <div className="flex gap-2 mt-1">
            <AlertTriangle className="h-4 w-4 text-red-500 flex-shrink-0 mt-0.5" />
            <span className="text-red-600 text-xs">{session.error}</span>
          </div>
        )}
      </div>
    </div>
  );
}
