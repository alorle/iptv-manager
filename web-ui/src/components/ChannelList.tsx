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
  const containerRef = useRef<HTMLElement>(null)
  const tableContainerRef = useRef<HTMLDivElement>(null)

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

  // Handle select all checkbox (selects all streams in filtered channels)
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      const allStreamIds = filteredChannels.flatMap((ch) => ch.streams.map((s) => s.acestream_id))
      setSelectedIds(new Set(allStreamIds))
    } else {
      setSelectedIds(new Set())
    }
  }

  // Handle individual checkbox for a stream
  const handleSelectOne = (id: string, checked: boolean) => {
    const newSelected = new Set(selectedIds)
    if (checked) {
      newSelected.add(id)
    } else {
      newSelected.delete(id)
    }
    setSelectedIds(newSelected)
  }

  // Handle row click (currently disabled since we now have channels with multiple streams)
  const handleRowClick = (channel: Channel, stream: (typeof channel.streams)[0]) => {
    // For now, we'll need to pass stream data to the parent
    // This will require updating the parent component to handle stream selection
    // For backward compatibility, we can create a synthetic Channel object
    const syntheticChannel: Channel = {
      name: stream.name,
      tvg_id: channel.tvg_id,
      tvg_logo: channel.tvg_logo,
      group_title: channel.group_title,
      streams: [stream],
    }
    onChannelSelect?.(syntheticChannel)
  }

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

  // Check if all streams in filtered channels are selected (must be before early returns)
  const allStreamIds = useMemo(
    () => new Set(filteredChannels.flatMap((ch) => ch.streams.map((s) => s.acestream_id))),
    [filteredChannels]
  )

  const allSelected =
    allStreamIds.size > 0 && Array.from(allStreamIds).every((id) => selectedIds.has(id))

  const someSelected =
    selectedIds.size > 0 &&
    !allSelected &&
    Array.from(allStreamIds).some((id) => selectedIds.has(id))

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
              <th scope="col">Name</th>
              <th scope="col">Group</th>
              <th scope="col">TVG-ID</th>
              <th className="status-column" scope="col">
                Status
              </th>
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
              filteredChannels.map((channel) =>
                channel.streams.map((stream, streamIndex) => (
                  <tr
                    key={stream.acestream_id}
                    className="channel-row"
                    onClick={() => handleRowClick(channel, stream)}
                    tabIndex={0}
                    role="button"
                    aria-label={`Edit ${stream.name}${!stream.enabled ? ' (disabled)' : ''}${stream.has_override ? ', has custom overrides' : ''}`}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        handleRowClick(channel, stream)
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
                        checked={selectedIds.has(stream.acestream_id)}
                        onChange={(e) => handleSelectOne(stream.acestream_id, e.target.checked)}
                        aria-label={`Select ${stream.name}`}
                      />
                    </td>
                    <td className="channel-name">
                      {stream.name}
                      {!stream.enabled && (
                        <span className="disabled-badge" aria-label="Stream is disabled">
                          Disabled
                        </span>
                      )}
                      {channel.streams.length > 1 && streamIndex === 0 && (
                        <span
                          className="stream-count"
                          title={`${channel.streams.length} streams in this channel`}
                        >
                          ({channel.streams.length})
                        </span>
                      )}
                    </td>
                    <td className="channel-group">{channel.group_title}</td>
                    <td className="channel-tvg-id">{channel.tvg_id || '-'}</td>
                    <td className="status-column">
                      {stream.has_override && (
                        <span
                          className="override-indicator"
                          title="Has custom overrides"
                          aria-label="Has custom overrides"
                          role="img"
                        >
                          ⚙
                        </span>
                      )}
                    </td>
                  </tr>
                ))
              )
            )}
          </tbody>
        </table>
      </div>

      <div className="channel-list-footer">
        <div id="search-results" className="footer-info" aria-live="polite">
          Showing {filteredChannels.reduce((sum, ch) => sum + ch.streams.length, 0)} stream(s) in{' '}
          {filteredChannels.length} channel(s) (Total:{' '}
          {channels.reduce((sum, ch) => sum + ch.streams.length, 0)} streams in {channels.length}{' '}
          channels)
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
