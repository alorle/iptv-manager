import { useState, useMemo } from 'react'
import type { Channel } from '../types'

export type EnabledFilter = 'all' | 'enabled' | 'disabled'

export interface ChannelFilters {
  searchText: string
  groupFilter: string
  enabledFilter: EnabledFilter
}

export interface ChannelFiltersResult {
  searchText: string
  groupFilter: string
  enabledFilter: EnabledFilter
  setSearchText: (text: string) => void
  setGroupFilter: (group: string) => void
  setEnabledFilter: (filter: EnabledFilter) => void
  filteredChannels: Channel[]
  uniqueGroups: string[]
}

/**
 * Custom hook for managing channel filtering logic
 * Handles search text, group filter, and enabled/disabled status filtering
 */
export function useChannelFilters(channels: Channel[]): ChannelFiltersResult {
  const [searchText, setSearchText] = useState('')
  const [groupFilter, setGroupFilter] = useState('')
  const [enabledFilter, setEnabledFilter] = useState<EnabledFilter>('enabled')

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

  return {
    searchText,
    groupFilter,
    enabledFilter,
    setSearchText,
    setGroupFilter,
    setEnabledFilter,
    filteredChannels,
    uniqueGroups,
  }
}
