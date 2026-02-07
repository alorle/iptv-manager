import { useState, useMemo } from 'react'
import type { Channel } from '../types'

export interface ChannelSelectionResult {
  selectedIds: Set<string>
  allSelected: boolean
  someSelected: boolean
  handleSelectAll: (checked: boolean) => void
  handleSelectOne: (id: string, checked: boolean) => void
  clearSelection: () => void
}

/**
 * Custom hook for managing channel selection logic
 * Handles individual and bulk selection of channels
 */
export function useChannelSelection(
  filteredChannels: Channel[],
  getChannelKey: (channel: Channel) => string
): ChannelSelectionResult {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  // Get all channel IDs from filtered list
  const allChannelIds = useMemo(
    () => new Set(filteredChannels.map((ch) => getChannelKey(ch))),
    [filteredChannels, getChannelKey]
  )

  // Check if all channels are selected
  const allSelected =
    allChannelIds.size > 0 && Array.from(allChannelIds).every((id) => selectedIds.has(id))

  // Check if some (but not all) channels are selected
  const someSelected =
    selectedIds.size > 0 &&
    !allSelected &&
    Array.from(allChannelIds).some((id) => selectedIds.has(id))

  // Handle select all checkbox (selects all channels in filtered list)
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      const allIds = filteredChannels.map((ch) => getChannelKey(ch))
      setSelectedIds(new Set(allIds))
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

  // Clear all selections
  const clearSelection = () => {
    setSelectedIds(new Set())
  }

  return {
    selectedIds,
    allSelected,
    someSelected,
    handleSelectAll,
    handleSelectOne,
    clearSelection,
  }
}
