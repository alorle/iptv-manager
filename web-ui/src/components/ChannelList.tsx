import { useState, useMemo, useEffect, useRef } from 'react'
import type { Channel } from '../types'
import { useChannels } from '../hooks/useChannels'
import { BulkEditModal } from './BulkEditModal'
import { ErrorDisplay } from './ErrorDisplay'
import { LoadingSpinner } from './LoadingSpinner'
import { bulkUpdateOverrides } from '../api/channels'
import type { useToast } from '../hooks/useToast'
import './ChannelList.css'

interface ChannelListProps {
  onChannelSelect?: (channel: Channel) => void
  refreshTrigger?: number
  toast: ReturnType<typeof useToast>
}

type EnabledFilter = 'all' | 'enabled' | 'disabled'

export function ChannelList({ onChannelSelect, refreshTrigger, toast }: ChannelListProps) {
  const { channels, loading, error, refetch } = useChannels(refreshTrigger)
  const [searchText, setSearchText] = useState('')
  const [groupFilter, setGroupFilter] = useState('')
  const [enabledFilter, setEnabledFilter] = useState<EnabledFilter>('enabled')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showBulkEditModal, setShowBulkEditModal] = useState(false)
  const [isScrolled, setIsScrolled] = useState(false)
  const [expandedChannelId, setExpandedChannelId] = useState<string | null>(null)
  const containerRef = useRef<HTMLElement>(null)
  const tableContainerRef = useRef<HTMLDivElement>(null)
  const expandedRowRef = useRef<HTMLTableRowElement>(null)

  // Detect scroll to show header shadow
  useEffect(() => {
    const tableContainer = tableContainerRef.current
    if (!tableContainer) return

    const handleScroll = () => {
      const scrollThreshold = 20
      setIsScrolled(tableContainer.scrollTop > scrollThreshold)
    }

    // Check initial scroll position
    handleScroll()

    tableContainer.addEventListener('scroll', handleScroll, { passive: true })
    return () => tableContainer.removeEventListener('scroll', handleScroll)
  }, [loading, channels.length])

  // Get unique group titles for the filter dropdown
  const uniqueGroups = useMemo(() => {
    const groups = new Set(channels.map((ch) => ch.group_title).filter(Boolean))
    return Array.from(groups).sort()
  }, [channels])

  // Generate unique channel key
  const getChannelKey = (channel: Channel) => {
    // Use tvg_id if available, otherwise use first stream's acestream_id
    return channel.tvg_id || channel.streams[0]?.acestream_id || channel.name
  }

  // Filter channels based on search, group filter, and enabled filter
  const filteredChannels = useMemo(() => {
    return channels.filter((channel) => {
      const matchesSearch =
        searchText === '' || channel.name.toLowerCase().includes(searchText.toLowerCase())

      const matchesGroup = groupFilter === '' || channel.group_title === groupFilter

      // Channel is considered enabled if at least one stream is enabled
      const hasEnabledStream = channel.streams.some((s) => s.enabled)
      const matchesEnabled =
        enabledFilter === 'all' ||
        (enabledFilter === 'enabled' && hasEnabledStream) ||
        (enabledFilter === 'disabled' && !hasEnabledStream)

      return matchesSearch && matchesGroup && matchesEnabled
    })
  }, [channels, searchText, groupFilter, enabledFilter])

  // Handle select all checkbox (selects all channels in filtered list)
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      const allChannelIds = filteredChannels.map((ch) => getChannelKey(ch))
      setSelectedIds(new Set(allChannelIds))
    } else {
      setSelectedIds(new Set())
    }
  }

  // Handle individual checkbox for a channel
  const handleSelectOne = (id: string, checked: boolean) => {
    const newSelected = new Set(selectedIds)
    if (checked) {
      newSelected.add(id)
    } else {
      newSelected.delete(id)
    }
    setSelectedIds(newSelected)
  }

  // Handle row click to expand/collapse channel
  const handleRowClick = (channel: Channel) => {
    const channelKey = getChannelKey(channel)
    if (expandedChannelId === channelKey) {
      setExpandedChannelId(null)
    } else {
      setExpandedChannelId(channelKey)
    }
  }

  // Scroll expanded channel into view
  useEffect(() => {
    if (expandedChannelId && expandedRowRef.current) {
      expandedRowRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
      })
    }
  }, [expandedChannelId])

  // Handle bulk edit submission
  const handleBulkEdit = async (field: string, value: string | boolean) => {
    try {
      const result = await bulkUpdateOverrides(Array.from(selectedIds), field, value)

      if (result.failed > 0) {
        toast.warning(`Updated ${result.updated} channel(s), but ${result.failed} failed`, 7000)
      } else {
        toast.success(`Successfully updated ${result.updated} channel(s)`)
      }

      // Clear selection and close modal
      setSelectedIds(new Set())
      setShowBulkEditModal(false)

      // Refresh channel list
      refetch()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update channels'
      toast.error(errorMessage)
      throw err
    }
  }

  // Check if all channels in filtered list are selected (must be before early returns)
  const allChannelIds = useMemo(
    () => new Set(filteredChannels.map((ch) => getChannelKey(ch))),
    [filteredChannels]
  )

  const allSelected =
    allChannelIds.size > 0 && Array.from(allChannelIds).every((id) => selectedIds.has(id))

  const someSelected =
    selectedIds.size > 0 &&
    !allSelected &&
    Array.from(allChannelIds).some((id) => selectedIds.has(id))

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
                  No channels found
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
                      onClick={() => handleRowClick(channel)}
                      tabIndex={0}
                      role="button"
                      aria-label={`${isExpanded ? 'Collapse' : 'Expand'} ${channel.name}${!hasEnabledStream ? ' (disabled)' : ''}${hasOverride ? ', has custom overrides' : ''}`}
                      aria-expanded={isExpanded}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          handleRowClick(channel)
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
                        {channel.streams.length > 1 ? (
                          <span className="stream-count">{channel.streams.length} streams</span>
                        ) : (
                          <span className="stream-count single">1 stream</span>
                        )}
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
