import { useState, useMemo } from 'react'
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

export function ChannelList({ onChannelSelect, refreshTrigger, toast }: ChannelListProps) {
  const { channels, loading, error, refetch } = useChannels(refreshTrigger)
  const [searchText, setSearchText] = useState('')
  const [groupFilter, setGroupFilter] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [showBulkEditModal, setShowBulkEditModal] = useState(false)

  // Get unique group titles for the filter dropdown
  const uniqueGroups = useMemo(() => {
    const groups = new Set(channels.map((ch) => ch.group_title).filter(Boolean))
    return Array.from(groups).sort()
  }, [channels])

  // Filter channels based on search and group filter
  const filteredChannels = useMemo(() => {
    return channels.filter((channel) => {
      const matchesSearch =
        searchText === '' || channel.name.toLowerCase().includes(searchText.toLowerCase())

      const matchesGroup = groupFilter === '' || channel.group_title === groupFilter

      return matchesSearch && matchesGroup
    })
  }, [channels, searchText, groupFilter])

  // Handle select all checkbox
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedIds(new Set(filteredChannels.map((ch) => ch.acestream_id)))
    } else {
      setSelectedIds(new Set())
    }
  }

  // Handle individual checkbox
  const handleSelectOne = (id: string, checked: boolean) => {
    const newSelected = new Set(selectedIds)
    if (checked) {
      newSelected.add(id)
    } else {
      newSelected.delete(id)
    }
    setSelectedIds(newSelected)
  }

  // Handle row click
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

  const allSelected =
    filteredChannels.length > 0 && filteredChannels.every((ch) => selectedIds.has(ch.acestream_id))

  const someSelected =
    selectedIds.size > 0 &&
    !allSelected &&
    filteredChannels.some((ch) => selectedIds.has(ch.acestream_id))

  return (
    <main className="channel-list-container" id="main-content" aria-labelledby="page-title">
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

      <div className="table-container" role="region" aria-label="Channel list">
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
              filteredChannels.map((channel) => (
                <tr
                  key={channel.acestream_id}
                  className="channel-row"
                  onClick={() => handleRowClick(channel)}
                  tabIndex={0}
                  role="button"
                  aria-label={`Edit ${channel.name}${!channel.enabled ? ' (disabled)' : ''}${channel.has_override ? ', has custom overrides' : ''}`}
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
                      checked={selectedIds.has(channel.acestream_id)}
                      onChange={(e) => handleSelectOne(channel.acestream_id, e.target.checked)}
                      aria-label={`Select ${channel.name}`}
                    />
                  </td>
                  <td className="channel-name">
                    {channel.name}
                    {!channel.enabled && (
                      <span className="disabled-badge" aria-label="Channel is disabled">
                        Disabled
                      </span>
                    )}
                  </td>
                  <td className="channel-group">{channel.group_title}</td>
                  <td className="channel-tvg-id">{channel.tvg_id || '-'}</td>
                  <td className="status-column">
                    {channel.has_override && (
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
            )}
          </tbody>
        </table>
      </div>

      <div className="channel-list-footer">
        <div id="search-results" className="footer-info" aria-live="polite">
          Showing {filteredChannels.length} of {channels.length} channels
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
