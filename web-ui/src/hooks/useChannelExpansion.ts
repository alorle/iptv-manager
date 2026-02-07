import { useState, useEffect, useRef } from 'react'
import type { Channel } from '../types'

export interface ChannelExpansionResult {
  expandedChannelId: string | null
  expandedRowRef: React.RefObject<HTMLTableRowElement | null>
  handleRowClick: (channel: Channel, getChannelKey: (channel: Channel) => string) => void
}

/**
 * Custom hook for managing channel expansion (accordion) logic
 * Handles expanding/collapsing channel details and scrolling into view
 */
export function useChannelExpansion(): ChannelExpansionResult {
  const [expandedChannelId, setExpandedChannelId] = useState<string | null>(null)
  const expandedRowRef = useRef<HTMLTableRowElement>(null)

  // Handle row click to expand/collapse channel
  const handleRowClick = (channel: Channel, getChannelKey: (channel: Channel) => string) => {
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

  return {
    expandedChannelId,
    expandedRowRef,
    handleRowClick,
  }
}
