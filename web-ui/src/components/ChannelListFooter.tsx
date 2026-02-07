import type { Channel } from '../types'

interface ChannelListFooterProps {
  filteredChannels: Channel[]
  totalChannels: Channel[]
}

export function ChannelListFooter({ filteredChannels, totalChannels }: ChannelListFooterProps) {
  const filteredStreamCount = filteredChannels.reduce((sum, ch) => sum + ch.streams.length, 0)
  const totalStreamCount = totalChannels.reduce((sum, ch) => sum + ch.streams.length, 0)

  return (
    <div className="channel-list-footer">
      <div id="search-results" className="footer-info" aria-live="polite">
        Showing {filteredChannels.length} channel(s) with {filteredStreamCount} stream(s) (Total:{' '}
        {totalChannels.length} channels with {totalStreamCount} streams)
      </div>
    </div>
  )
}
