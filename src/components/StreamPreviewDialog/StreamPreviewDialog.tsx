import { useState } from 'react';
import { Play, ExternalLink, Copy, Check } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { paths, components } from '../../lib/api/v1';
import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";

const fetchClient = createFetchClient<paths>({
  baseUrl: "/api/",
});
const $api = createClient(fetchClient);

type Stream = components["schemas"]["Stream"];

interface StreamPreviewDialogProps {
  stream: Stream;
  channelTitle?: string;
}

export default function StreamPreviewDialog({
  stream,
  channelTitle,
}: StreamPreviewDialogProps) {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);

  // Fetch config to get acestream URL
  const { data: config, isLoading: configLoading } = $api.useQuery(
    "get",
    "/config",
  );

  // Build stream URL
  const streamUrl = config?.acestream_url
    ? `${config.acestream_url}?id=${stream.acestream_id}&network-caching=${stream.network_caching}`
    : null;

  const copyToClipboard = async () => {
    if (streamUrl) {
      await navigator.clipboard.writeText(streamUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const openInVLC = () => {
    if (streamUrl) {
      window.location.href = `vlc://${streamUrl}`;
    }
  };

  return (
    <>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setOpen(true)}
        disabled={configLoading || !streamUrl}
        title="Stream info & playback options"
      >
        <Play className="h-4 w-4" />
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              Stream Info & Playback
              {channelTitle && ` - ${channelTitle}`}
              {stream.quality && ` [${stream.quality}]`}
            </DialogTitle>
            <DialogDescription>
              Stream information and quick actions for external playback.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            {/* Quick Actions */}
            {streamUrl && (
              <div className="space-y-2">
                <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  Quick Actions
                </h3>
                <div className="flex gap-2">
                  <Button
                    onClick={openInVLC}
                    className="flex-1"
                    variant="default"
                  >
                    <ExternalLink className="h-4 w-4 mr-2" />
                    Open in VLC
                  </Button>
                  <Button
                    onClick={copyToClipboard}
                    className="flex-1"
                    variant="outline"
                  >
                    {copied ? (
                      <>
                        <Check className="h-4 w-4 mr-2" />
                        Copied!
                      </>
                    ) : (
                      <>
                        <Copy className="h-4 w-4 mr-2" />
                        Copy URL
                      </>
                    )}
                  </Button>
                </div>
              </div>
            )}

            {/* Stream Information */}
            <div className="space-y-3 text-sm border-t pt-4">
              <div>
                <span className="font-semibold text-gray-700 dark:text-gray-300">Acestream ID:</span>
                <p className="font-mono text-xs bg-gray-100 dark:bg-gray-800 p-2 rounded mt-1 break-all">
                  {stream.acestream_id}
                </p>
              </div>

              <div className="grid grid-cols-2 gap-3">
                {stream.quality && (
                  <div>
                    <span className="font-semibold text-gray-700 dark:text-gray-300">Quality:</span>
                    <p className="mt-1">{stream.quality}</p>
                  </div>
                )}

                <div>
                  <span className="font-semibold text-gray-700 dark:text-gray-300">Network Caching:</span>
                  <p className="mt-1">{stream.network_caching}ms</p>
                </div>
              </div>

              {stream.tags && stream.tags.length > 0 && (
                <div>
                  <span className="font-semibold text-gray-700 dark:text-gray-300">Tags:</span>
                  <div className="flex gap-2 mt-1 flex-wrap">
                    {stream.tags.map((tag, idx) => (
                      <span
                        key={idx}
                        className="px-2 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded text-xs font-medium"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {streamUrl && (
                <div>
                  <span className="font-semibold text-gray-700 dark:text-gray-300">Stream URL:</span>
                  <p className="font-mono text-xs bg-gray-100 dark:bg-gray-800 p-2 rounded mt-1 break-all text-gray-600 dark:text-gray-400">
                    {streamUrl}
                  </p>
                </div>
              )}
            </div>

            {/* Playback Instructions */}
            <div className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-lg p-4 text-sm">
              <p className="font-semibold text-blue-900 dark:text-blue-100 mb-3">How to play this stream:</p>
              <ul className="list-decimal list-inside space-y-2 text-blue-800 dark:text-blue-200">
                <li>
                  <strong>VLC Media Player (Recommended):</strong>
                  <ul className="list-disc list-inside ml-6 mt-1 space-y-1 text-sm">
                    <li>Click <strong>"Open in VLC"</strong> button above for instant playback</li>
                    <li>Or open VLC → Media → Open Network Stream → paste URL</li>
                  </ul>
                </li>
                <li>
                  <strong>IPTV Players:</strong>
                  <ul className="list-disc list-inside ml-6 mt-1 space-y-1 text-sm">
                    <li>Use the M3U playlist in Kodi, TiviMate, Perfect Player, etc.</li>
                    <li>Download playlist from the home page</li>
                  </ul>
                </li>
                <li>
                  <strong>Any Media Player:</strong>
                  <ul className="list-disc list-inside ml-6 mt-1 space-y-1 text-sm">
                    <li>Click <strong>"Copy URL"</strong> and paste it into your preferred player</li>
                  </ul>
                </li>
              </ul>
              <p className="text-blue-700 dark:text-blue-300 mt-3 pt-3 border-t border-blue-200 dark:border-blue-700">
                <strong>Note:</strong> Acestream uses P2P technology. Initial buffering may take 10-30 seconds while establishing connections.
              </p>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}
