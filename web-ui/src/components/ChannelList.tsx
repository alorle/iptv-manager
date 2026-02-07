import { useState, useRef } from 'react'
import type { Channel } from '../types'
import { useChannels } from '../hooks/useChannels'
import { useChannelFilters } from '../hooks/useChannelFilters'
import { useChannelSelection } from '../hooks/useChannelSelection'
import { useChannelExpansion } from '../hooks/useChannelExpansion'
import { useScrollDetection } from '../hooks/useScrollDetection'
import { useChannelBulkEdit } from '../hooks/useChannelBulkEdit'
import { BulkEditModal } from './BulkEditModal'
import { ErrorDisplay } from './ErrorDisplay'
import { LoadingSpinner } from './LoadingSpinner'
import type { useToast } from '../hooks/useToast'
import './ChannelList.css'

interface ChannelListProps {
  onChannelSelect?: (channel: Channel) => void
  refreshTrigger?: number
  toast: ReturnType<typeof useToast>
}

export function ChannelList({ onChannelSelect, refreshTrigger, toast }: ChannelListProps) {
  const { channels, loading, error, refetch } = useChannels(refreshTrigger)
  const [showBulkEditModal, setShowBulkEditModal] = useState(false)
  const containerRef = useRef<HTMLElement>(null)

  // Generate unique channel key
  const getChannelKey = (channel: Channel) => {
    // Use tvg_id if available, otherwise use first stream's acestream_id
    return channel.tvg_id || channel.streams[0]?.acestream_id || channel.name
  }

  // Custom hooks for logic separation
  const {
    searchText,
    groupFilter,
    enabledFilter,
    setSearchText,
    setGroupFilter,
    setEnabledFilter,
    filteredChannels,
    uniqueGroups,
  } = useChannelFilters(channels)

  const {
    selectedIds,
    allSelected,
    someSelected,
    handleSelectAll,
    handleSelectOne,
    clearSelection,
  } = useChannelSelection(filteredChannels, getChannelKey)

  const { expandedChannelId, expandedRowRef, handleRowClick } = useChannelExpansion()

  const { isScrolled, tableContainerRef } = useScrollDetection(20, [loading, channels.length])

  const { handleBulkEdit: bulkEditOperation } = useChannelBulkEdit(toast, () => {
    clearSelection()
    setShowBulkEditModal(false)
    refetch()
  })

  // Wrapper to pass selectedIds to bulk edit operation
  const handleBulkEdit = async (field: string, value: string | boolean) => {
    await bulkEditOperation(selectedIds, field, value)
  }

  if (loading) {
    return (
      <div className="channel-list-container">
        <LoadingSpinner size="large" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="channel-list-container">
        <ErrorDisplay error={error} onRetry={refetch} title="Failed to load channels" />
      </div>
    )
  }

  return (
    <main
      ref={containerRef}
      className={`channel-list-container${isScrolled ? ' scrolled' : ''}`}
      id="main-content"
      aria-labelledby="page-title"
    >
      <div className="channel-list-header">
        <h1 id="page-title">Channel Management</h1>
        <div className="filters" role="search" aria-label="Filter channels">
          <label htmlFor="channel-search" className="visually-hidden">
            Search channels
          </label>
          <input
            id="channel-search"
            type="search"
            className="search-input"
            placeholder="Search channels..."
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            aria-describedby="search-results"
          />
          <label htmlFor="group-filter" className="visually-hidden">
            Filter by group
          </label>
          <select
            id="group-filter"
            className="group-filter"
            value={groupFilter}
            onChange={(e) => setGroupFilter(e.target.value)}
          >
            <option value="">All Groups</option>
            {uniqueGroups.map((group) => (
              <option key={group} value={group}>
                {group}
              </option>
            ))}
          </select>
          <div className="enabled-filter" role="group" aria-label="Filter by enabled status">
            <button
              type="button"
              className={`filter-button ${enabledFilter === 'all' ? 'active' : ''}`}
              onClick={() => setEnabledFilter('all')}
              aria-pressed={enabledFilter === 'all'}
            >
              All
            </button>
            <button
              type="button"
              className={`filter-button ${enabledFilter === 'enabled' ? 'active' : ''}`}
              onClick={() => setEnabledFilter('enabled')}
              aria-pressed={enabledFilter === 'enabled'}
            >
              Enabled
            </button>
            <button
              type="button"
              className={`filter-button ${enabledFilter === 'disabled' ? 'active' : ''}`}
              onClick={() => setEnabledFilter('disabled')}
              aria-pressed={enabledFilter === 'disabled'}
            >
              Disabled
            </button>
          </div>
        </div>
        {selectedIds.size > 0 && (
          <div className="bulk-actions" role="toolbar" aria-label="Bulk actions">
            <div className="selection-info" aria-live="polite" aria-atomic="true">
              {selectedIds.size} channel(s) selected
            </div>
            <button
              type="button"
              className="button button-primary"
              onClick={() => setShowBulkEditModal(true)}
            >
              Bulk Edit
            </button>
          </div>
        )}
      </div>

      <div
        ref={tableContainerRef}
        className="table-container"
        role="region"
        aria-label="Channel list"
      >
        <table className="channel-table" aria-describedby="search-results">
          <thead>
            <tr>
              <th className="checkbox-column" scope="col">
                <input
                  type="checkbox"
                  checked={allSelected}
                  ref={(el) => {
                    if (el) el.indeterminate = someSelected
                  }}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                  aria-label={allSelected ? 'Deselect all channels' : 'Select all channels'}
                />
              </th>
              <th className="logo-column" scope="col">
                Logo
              </th>
              <th scope="col">Name</th>
              <th scope="col">Group</th>
              <th scope="col">Streams</th>
            </tr>
          </thead>
          <tbody>
            {filteredChannels.length === 0 ? (
              <tr>
                <td colSpan={5} className="empty-state">
                  {enabledFilter === 'enabled' && channels.length > 0
                    ? 'No channels with enabled streams found'
                    : enabledFilter === 'disabled' && channels.length > 0
                      ? 'No channels with all streams disabled found'
                      : searchText || groupFilter
                        ? 'No channels match the current filters'
                        : 'No channels found'}
                </td>
              </tr>
            ) : (
              filteredChannels.map((channel) => {
                const channelKey = getChannelKey(channel)
                const hasEnabledStream = channel.streams.some((s) => s.enabled)
                const hasOverride = channel.streams.some((s) => s.has_override)
                const isExpanded = expandedChannelId === channelKey
                return (
                  <>
                    <tr
                      key={channelKey}
                      ref={isExpanded ? expandedRowRef : null}
                      className={`channel-row${isExpanded ? ' expanded' : ''}`}
                      onClick={() => handleRowClick(channel, getChannelKey)}
                      tabIndex={0}
                      role="button"
                      aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${channel.name}${!hasEnabledStream ? ' (disabled)' : ''}${hasOverride ? ', has custom overrides' : ''}`}
                      aria-expanded={isExpanded}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          handleRowClick(channel, getChannelKey)
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
                          checked={selectedIds.has(channelKey)}
                          onChange={(e) => handleSelectOne(channelKey, e.target.checked)}
                          aria-label={`Select ${channel.name}`}
                        />
                      </td>
                      <td className="logo-column">
                        {channel.tvg_logo ? (
                          <img
                            src={channel.tvg_logo}
                            alt={`${channel.name} logo`}
                            className="channel-logo"
                            onError={(e) => {
                              e.currentTarget.style.display = 'none'
                              const placeholder = e.currentTarget.nextElementSibling as HTMLElement
                              if (placeholder) placeholder.style.display = 'flex'
                            }}
                          />
                        ) : null}
                        <div
                          className="channel-logo-placeholder"
                          style={{ display: channel.tvg_logo ? 'none' : 'flex' }}
                          aria-label="No logo available"
                        >
                          <svg
                            width="24"
                            height="24"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                          >
                            <rect x="2" y="7" width="20" height="15" rx="2" ry="2" />
                            <polyline points="17 2 12 7 7 2" />
                          </svg>
                        </div>
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
                          {!hasEnabledStream && (
                            <span className="disabled-badge" aria-label="Channel is disabled">
                              Disabled
                            </span>
                          )}
                          {hasOverride && (
                            <span
                              className="override-indicator-inline"
                              title="Has custom overrides"
                              aria-label="Has custom overrides"
                            >
                              ⚙
                            </span>
                          )}
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
                    {isExpanded && (
                      <tr key={`${channelKey}-streams`} className="stream-expansion">
                        <td colSpan={5} className="stream-expansion-cell">
                          <div className="stream-list">
                            <div className="stream-list-header">
                              <h3>Streams for {channel.name}</h3>
                              <button
                                type="button"
                                className="button button-small"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  onChannelSelect?.(channel)
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
                                    {!stream.enabled && (
                                      <span
                                        className="disabled-badge"
                                        aria-label="Stream is disabled"
                                      >
                                        Disabled
                                      </span>
                                    )}
                                    {stream.has_override && (
                                      <span
                                        className="override-indicator-inline"
                                        title="Has custom overrides"
                                        aria-label="Has custom overrides"
                                      >
                                        ⚙
                                      </span>
                                    )}
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
                    )}
                  </>
                )
              })
            )}
          </tbody>
        </table>
      </div>

      <div className="channel-list-footer">
        <div id="search-results" className="footer-info" aria-live="polite">
          Showing {filteredChannels.length} channel(s) with{' '}
          {filteredChannels.reduce((sum, ch) => sum + ch.streams.length, 0)} stream(s) (Total:{' '}
          {channels.length} channels with {channels.reduce((sum, ch) => sum + ch.streams.length, 0)}{' '}
          streams)
        </div>
      </div>

      {showBulkEditModal && (
        <BulkEditModal
          selectedCount={selectedIds.size}
          onClose={() => setShowBulkEditModal(false)}
          onSubmit={handleBulkEdit}
        />
      )}
    </main>
  )
}
