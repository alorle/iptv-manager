import "./ChannelListItem.css";
import { components } from '../../lib/api/v1';

type Channel = components["schemas"]["Channel"];
type Stream = components["schemas"]["Stream"];

export default function ChannelListItem({ index, channel }: { index: number, channel: Channel }) {

    return (
        <div style={{ marginBottom: '2rem', padding: '1rem', border: '1px solid #ccc' }}>
            <h1>#{index} - {channel.title}</h1>
            <p><strong>Guide ID:</strong> {channel.guide_id}</p>
            {channel.logo && <p><strong>Logo:</strong> {channel.logo}</p>}
            <p><strong>Group:</strong> {channel.group_title}</p>

            <h3>Streams ({channel.streams.length})</h3>
            <div style={{ marginLeft: '1rem' }}>
                {channel.streams.map((stream: Stream, streamIndex: number) => (
                    <div key={stream.id} style={{ marginBottom: '1rem', padding: '0.5rem', backgroundColor: '#f5f5f5' }}>
                        <p><strong>Stream #{streamIndex + 1}</strong></p>
                        <p><code>{stream.acestream_id}</code></p>
                        {stream.quality && <p><strong>Quality:</strong> {stream.quality}</p>}
                        {stream.tags && stream.tags.length > 0 && (
                            <p><strong>Tags:</strong> {stream.tags.join(", ")}</p>
                        )}
                        <p><strong>Network Caching:</strong> {stream.network_caching}ms</p>
                    </div>
                ))}
            </div>
        </div>
    )
}