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

  // Handle select all checkbox (selects all channels in filtered list)
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      const allChannelIds = filteredChannels.map((ch) => ch.tvg_id)
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

  // Handle row click to edit channel
  const handleRowClick = (channel: Channel) => {
    onChannelSelect?.(channel)
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

  // Check if all channels in filtered list are selected (must be before early returns)
  const allChannelIds = useMemo(
    () => new Set(filteredChannels.map((ch) => ch.tvg_id)),
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
                const hasEnabledStream = channel.streams.some((s) => s.enabled)
                const hasOverride = channel.streams.some((s) => s.has_override)
                return (
                  <tr
                    key={channel.tvg_id}
                    className="channel-row"
                    onClick={() => handleRowClick(channel)}
                    tabIndex={0}
                    role="button"
                    aria-label={`Edit ${channel.name}${!hasEnabledStream ? ' (disabled)' : ''}${hasOverride ? ', has custom overrides' : ''}`}
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
                        checked={selectedIds.has(channel.tvg_id)}
                        onChange={(e) => handleSelectOne(channel.tvg_id, e.target.checked)}
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
