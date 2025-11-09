import "./ChannelListItem.css";
import { components } from '../../lib/api/v1';

type Channel = components["schemas"]["Channel"];

export default function ChannelListItem({ index, channel }: { index: number, channel: Channel }) {
    
    return (<>
    <h1>#{index} - {channel.name}</h1>
    <code>{channel.acestream_id}</code>
    {channel.category && <p>{channel.category}</p>}
    {channel.epg_id && <p>{channel.epg_id}</p>}
    {channel.quality && <p>{channel.quality}</p>}
    {channel.tags && <p>{channel.tags.join(", ")}</p>}
    </>)
}