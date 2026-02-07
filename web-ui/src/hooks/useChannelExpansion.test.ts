import { describe, it, expect, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useChannelExpansion } from './useChannelExpansion'
import type { Channel } from '../types'

const mockChannel: Channel = {
  name: 'BBC One',
  tvg_id: 'bbc1.uk',
  tvg_logo: 'https://example.com/bbc1.png',
  group_title: 'UK',
  streams: [
    {
      acestream_id: 'abc123',
      name: 'BBC One HD',
      tvg_name: 'BBC One',
      source: 'elcano',
      enabled: true,
      has_override: false,
    },
  ],
}

const mockChannel2: Channel = {
  name: 'ITV',
  tvg_id: 'itv.uk',
  tvg_logo: 'https://example.com/itv.png',
  group_title: 'UK',
  streams: [
    {
      acestream_id: 'def456',
      name: 'ITV HD',
      tvg_name: 'ITV',
      source: 'elcano',
      enabled: true,
      has_override: false,
    },
  ],
}

const getChannelKey = (channel: Channel) => channel.name

describe('useChannelExpansion', () => {
  it('should initialize with no expanded channel', () => {
    const { result } = renderHook(() => useChannelExpansion())

    expect(result.current.expandedChannelId).toBeNull()
    expect(result.current.expandedRowRef.current).toBeNull()
  })

  it('should expand a channel when clicked', () => {
    const { result } = renderHook(() => useChannelExpansion())

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })

    expect(result.current.expandedChannelId).toBe('BBC One')
  })

  it('should collapse a channel when clicked again', () => {
    const { result } = renderHook(() => useChannelExpansion())

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })

    expect(result.current.expandedChannelId).toBe('BBC One')

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })

    expect(result.current.expandedChannelId).toBeNull()
  })

  it('should switch expanded channel when clicking a different channel', () => {
    const { result } = renderHook(() => useChannelExpansion())

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })

    expect(result.current.expandedChannelId).toBe('BBC One')

    act(() => {
      result.current.handleRowClick(mockChannel2, getChannelKey)
    })

    expect(result.current.expandedChannelId).toBe('ITV')
  })

  it('should call scrollIntoView when a channel is expanded', () => {
    const { result } = renderHook(() => useChannelExpansion())

    const mockScrollIntoView = vi.fn()
    const mockElement = {
      scrollIntoView: mockScrollIntoView,
    } as unknown as HTMLTableRowElement

    // Manually set the ref
    act(() => {
      if (result.current.expandedRowRef.current !== undefined) {
        result.current.expandedRowRef.current = mockElement
      }
    })

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })

    // The effect should trigger after expandedChannelId changes
    expect(mockScrollIntoView).toHaveBeenCalledWith({
      behavior: 'smooth',
      block: 'nearest',
    })
  })

  it('should handle custom getChannelKey function', () => {
    const customGetKey = (channel: Channel) => channel.tvg_id

    const { result } = renderHook(() => useChannelExpansion())

    act(() => {
      result.current.handleRowClick(mockChannel, customGetKey)
    })

    expect(result.current.expandedChannelId).toBe('bbc1.uk')
  })

  it('should toggle between multiple channels correctly', () => {
    const { result } = renderHook(() => useChannelExpansion())

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })
    expect(result.current.expandedChannelId).toBe('BBC One')

    act(() => {
      result.current.handleRowClick(mockChannel2, getChannelKey)
    })
    expect(result.current.expandedChannelId).toBe('ITV')

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })
    expect(result.current.expandedChannelId).toBe('BBC One')

    act(() => {
      result.current.handleRowClick(mockChannel, getChannelKey)
    })
    expect(result.current.expandedChannelId).toBeNull()
  })
})
