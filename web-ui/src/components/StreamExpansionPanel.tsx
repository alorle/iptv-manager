import type { Channel } from '../types'
import { StatusBadge } from './StatusBadge'

interface StreamExpansionPanelProps {
  channel: Channel
  onEditChannel: (channel: Channel) => void
}

export function StreamExpansionPanel({ channel, onEditChannel }: StreamExpansionPanelProps) {
  return (
    <tr className="stream-expansion">
      <td colSpan={5} className="stream-expansion-cell">
        <div className="stream-list">
          <div className="stream-list-header">
            <h3>Streams for {channel.name}</h3>
            <button
              type="button"
              className="button button-small"
              onClick={(e) => {
                e.stopPropagation()
                onEditChannel(channel)
              }}
            >
              Edit Channel
            </button>
          </div>
          {channel.streams.map((stream) => (
            <div key={stream.acestream_id} className="stream-item">
              <div className="stream-info">
                <div className="stream-name">
                  <span className="stream-name-text">{stream.name}</span>
                  {!stream.enabled && <StatusBadge type="disabled" label="Stream is disabled" />}
                  {stream.has_override && <StatusBadge type="override" />}
                </div>
                <div className="stream-meta">
                  <span className="stream-source">{stream.source}</span>
                  <span className="stream-id">{stream.acestream_id}</span>
                </div>
              </div>
              <div className="stream-actions">
                <button
                  type="button"
                  className="button button-small button-secondary"
                  onClick={(e) => {
                    e.stopPropagation()
                    window.open(`/stream?id=${stream.acestream_id}`, '_blank')
                  }}
                  disabled={!stream.enabled}
                >
                  Play
                </button>
              </div>
            </div>
          ))}
        </div>
      </td>
    </tr>
  )
}
