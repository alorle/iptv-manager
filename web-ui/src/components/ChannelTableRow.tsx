import type { Channel } from '../types'
import { ChannelLogo } from './ChannelLogo'
import { StatusBadge } from './StatusBadge'

interface ChannelTableRowProps {
  channel: Channel
  channelKey: string
  isExpanded: boolean
  isSelected: boolean
  enabledFilter: 'all' | 'enabled' | 'disabled'
  expandedRowRef: React.RefObject<HTMLTableRowElement> | null
  onRowClick: () => void
  onSelectOne: (key: string, checked: boolean) => void
}

export function ChannelTableRow({
  channel,
  channelKey,
  isExpanded,
  isSelected,
  enabledFilter,
  expandedRowRef,
  onRowClick,
  onSelectOne,
}: ChannelTableRowProps) {
  const hasEnabledStream = channel.streams.some((s) => s.enabled)
  const hasOverride = channel.streams.some((s) => s.has_override)

  return (
    <tr
      key={channelKey}
      ref={isExpanded ? expandedRowRef : null}
      className={`channel-row${isExpanded ? ' expanded' : ''}`}
      onClick={onRowClick}
      tabIndex={0}
      role="button"
      aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${channel.name}${!hasEnabledStream ? ' (disabled)' : ''}${hasOverride ? ', has custom overrides' : ''}`}
      aria-expanded={isExpanded}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onRowClick()
        }
      }}
    >
      <td
        className="checkbox-column"
        onClick={(e) => e.stopPropagation()}
        onKeyDown={(e) => e.stopPropagation()}
      >
        <input
          type="checkbox"
          checked={isSelected}
          onChange={(e) => onSelectOne(channelKey, e.target.checked)}
          aria-label={`Select ${channel.name}`}
        />
      </td>
      <td className="logo-column">
        <ChannelLogo logo={channel.tvg_logo} channelName={channel.name} />
      </td>
      <td className="channel-name">
        <div className="channel-name-content">
          <svg
            className={`chevron-icon${isExpanded ? ' expanded' : ''}`}
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            aria-hidden="true"
          >
            <polyline points="9 18 15 12 9 6" />
          </svg>
          <span className="channel-name-text">{channel.name}</span>
          {!hasEnabledStream && <StatusBadge type="disabled" label="Channel is disabled" />}
          {hasOverride && <StatusBadge type="override" />}
        </div>
      </td>
      <td className="channel-group">
        <span className="group-badge">{channel.group_title}</span>
      </td>
      <td className="stream-count-cell">
        {(() => {
          const enabledCount = channel.streams.filter((s) => s.enabled).length
          const totalCount = channel.streams.length
          if (totalCount === 1) {
            return <span className="stream-count single">1 stream</span>
          }
          if (enabledFilter !== 'all' && enabledCount !== totalCount) {
            return (
              <span className="stream-count">
                {enabledCount}/{totalCount} streams
              </span>
            )
          }
          return <span className="stream-count">{totalCount} streams</span>
        })()}
      </td>
    </tr>
  )
}
