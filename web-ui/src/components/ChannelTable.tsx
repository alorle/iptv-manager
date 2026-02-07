import type { Channel } from '../types'
import { ChannelTableHeader } from './ChannelTableHeader'
import { ChannelTableRow } from './ChannelTableRow'
import { StreamExpansionPanel } from './StreamExpansionPanel'
import { EmptyState } from './EmptyState'

interface ChannelTableProps {
  channels: Channel[]
  filteredChannels: Channel[]
  enabledFilter: 'all' | 'enabled' | 'disabled'
  searchText: string
  groupFilter: string
  allSelected: boolean
  someSelected: boolean
  selectedIds: Set<string>
  expandedChannelId: string | null
  expandedRowRef: React.RefObject<HTMLTableRowElement>
  tableContainerRef: React.RefObject<HTMLDivElement>
  getChannelKey: (channel: Channel) => string
  onSelectAll: (checked: boolean) => void
  onSelectOne: (key: string, checked: boolean) => void
  onRowClick: (channel: Channel, getKey: (channel: Channel) => string) => void
  onChannelSelect?: (channel: Channel) => void
}

export function ChannelTable({
  channels,
  filteredChannels,
  enabledFilter,
  searchText,
  groupFilter,
  allSelected,
  someSelected,
  selectedIds,
  expandedChannelId,
  expandedRowRef,
  tableContainerRef,
  getChannelKey,
  onSelectAll,
  onSelectOne,
  onRowClick,
  onChannelSelect,
}: ChannelTableProps) {
  return (
    <div
      ref={tableContainerRef}
      className="table-container"
      role="region"
      aria-label="Channel list"
    >
      <table className="channel-table" aria-describedby="search-results">
        <ChannelTableHeader
          allSelected={allSelected}
          someSelected={someSelected}
          onSelectAll={onSelectAll}
        />
        <tbody>
          {filteredChannels.length === 0 ? (
            <EmptyState
              enabledFilter={enabledFilter}
              totalChannelsCount={channels.length}
              searchText={searchText}
              groupFilter={groupFilter}
            />
          ) : (
            filteredChannels.map((channel) => {
              const channelKey = getChannelKey(channel)
              const isExpanded = expandedChannelId === channelKey
              return (
                <>
                  <ChannelTableRow
                    key={channelKey}
                    channel={channel}
                    channelKey={channelKey}
                    isExpanded={isExpanded}
                    isSelected={selectedIds.has(channelKey)}
                    enabledFilter={enabledFilter}
                    expandedRowRef={isExpanded ? expandedRowRef : null}
                    onRowClick={() => onRowClick(channel, getChannelKey)}
                    onSelectOne={onSelectOne}
                  />
                  {isExpanded && (
                    <StreamExpansionPanel
                      key={`${channelKey}-streams`}
                      channel={channel}
                      onEditChannel={() => onChannelSelect?.(channel)}
                    />
                  )}
                </>
              )
            })
          )}
        </tbody>
      </table>
    </div>
  )
}
