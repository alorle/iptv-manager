import './ChannelsList.css'
import { paths } from '../../lib/api/v1';
import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";
import ChannelListItem from '../ChannelListItem/ChannelListItem';

const fetchClient = createFetchClient<paths>({
  baseUrl: "/api/",
});
const $api = createClient(fetchClient);

function ChannelsList() {
  const { data: channels, error, isLoading } = $api.useQuery(
    "get",
    "/channels",
  );

  if (isLoading || !channels) return "Loading...";

  if (error) return `An error occured: ${error.message}`;

  return <div>{channels.map((channel, index) => (
    <ChannelListItem key={channel.id} channel={channel} index={index} />
  ))}</div>;
}

export default ChannelsList
