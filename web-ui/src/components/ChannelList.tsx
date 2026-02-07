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
import { ChannelFilters } from './ChannelFilters'
import { BulkActionsBar } from './BulkActionsBar'
import { ChannelTable } from './ChannelTable'
import { ChannelListFooter } from './ChannelListFooter'
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
        <ChannelFilters
          searchText={searchText}
          groupFilter={groupFilter}
          enabledFilter={enabledFilter}
          uniqueGroups={uniqueGroups}
          onSearchChange={setSearchText}
          onGroupFilterChange={setGroupFilter}
          onEnabledFilterChange={setEnabledFilter}
        />
        <BulkActionsBar
          selectedCount={selectedIds.size}
          onBulkEdit={() => setShowBulkEditModal(true)}
        />
      </div>

      <ChannelTable
        channels={channels}
        filteredChannels={filteredChannels}
        enabledFilter={enabledFilter}
        searchText={searchText}
        groupFilter={groupFilter}
        allSelected={allSelected}
        someSelected={someSelected}
        selectedIds={selectedIds}
        expandedChannelId={expandedChannelId}
        expandedRowRef={expandedRowRef}
        tableContainerRef={tableContainerRef}
        getChannelKey={getChannelKey}
        onSelectAll={handleSelectAll}
        onSelectOne={handleSelectOne}
        onRowClick={handleRowClick}
        onChannelSelect={onChannelSelect}
      />

      <ChannelListFooter filteredChannels={filteredChannels} totalChannels={channels} />

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
