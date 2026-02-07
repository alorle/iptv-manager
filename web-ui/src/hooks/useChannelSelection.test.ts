import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useChannelSelection } from './useChannelSelection'
import type { Channel } from '../types'

const mockChannels: Channel[] = [
  {
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
  },
  {
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
  },
  {
    name: 'TVE',
    tvg_id: 'tve.es',
    tvg_logo: 'https://example.com/tve.png',
    group_title: 'Spain',
    streams: [
      {
        acestream_id: 'ghi789',
        name: 'TVE HD',
        tvg_name: 'TVE',
        source: 'newera',
        enabled: true,
        has_override: false,
      },
    ],
  },
]

const getChannelKey = (channel: Channel) => channel.name

describe('useChannelSelection', () => {
  it('should initialize with no selections', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    expect(result.current.selectedIds.size).toBe(0)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(false)
  })

  it('should select a single channel', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectOne('BBC One', true)
    })

    expect(result.current.selectedIds.has('BBC One')).toBe(true)
    expect(result.current.selectedIds.size).toBe(1)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(true)
  })

  it('should deselect a single channel', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectOne('BBC One', true)
      result.current.handleSelectOne('BBC One', false)
    })

    expect(result.current.selectedIds.has('BBC One')).toBe(false)
    expect(result.current.selectedIds.size).toBe(0)
    expect(result.current.someSelected).toBe(false)
  })

  it('should select all channels', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectAll(true)
    })

    expect(result.current.selectedIds.size).toBe(3)
    expect(result.current.allSelected).toBe(true)
    expect(result.current.someSelected).toBe(false)
  })

  it('should deselect all channels', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectAll(true)
      result.current.handleSelectAll(false)
    })

    expect(result.current.selectedIds.size).toBe(0)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(false)
  })

  it('should indicate someSelected when some but not all channels are selected', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectOne('BBC One', true)
    })

    act(() => {
      result.current.handleSelectOne('ITV', true)
    })

    expect(result.current.selectedIds.size).toBe(2)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(true)
  })

  it('should clear all selections', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectAll(true)
      result.current.clearSelection()
    })

    expect(result.current.selectedIds.size).toBe(0)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(false)
  })

  it('should update allSelected when filtered channels change', () => {
    const { result, rerender } = renderHook(
      ({ channels }) => useChannelSelection(channels, getChannelKey),
      {
        initialProps: { channels: mockChannels },
      }
    )

    act(() => {
      result.current.handleSelectAll(true)
    })

    expect(result.current.allSelected).toBe(true)

    // Simulate filtering that reduces the channel list
    const filteredChannels = mockChannels.slice(0, 2)
    rerender({ channels: filteredChannels })

    // Should still be all selected because all filtered channels are selected
    expect(result.current.allSelected).toBe(true)
  })

  it('should handle empty channel list', () => {
    const { result } = renderHook(() => useChannelSelection([], getChannelKey))

    expect(result.current.selectedIds.size).toBe(0)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(false)

    act(() => {
      result.current.handleSelectAll(true)
    })

    expect(result.current.selectedIds.size).toBe(0)
    expect(result.current.allSelected).toBe(false)
  })

  it('should maintain selections when channels are added', () => {
    const { result, rerender } = renderHook(
      ({ channels }) => useChannelSelection(channels, getChannelKey),
      {
        initialProps: { channels: mockChannels },
      }
    )

    act(() => {
      result.current.handleSelectOne('BBC One', true)
    })

    expect(result.current.selectedIds.has('BBC One')).toBe(true)

    const newChannels: Channel[] = [
      ...mockChannels,
      {
        name: 'Canal+',
        tvg_id: 'canalplus.fr',
        tvg_logo: 'https://example.com/canalplus.png',
        group_title: 'France',
        streams: [
          {
            acestream_id: 'jkl012',
            name: 'Canal+ HD',
            tvg_name: 'Canal+',
            source: 'elcano',
            enabled: true,
            has_override: false,
          },
        ],
      },
    ]

    rerender({ channels: newChannels })

    // Original selection should be maintained
    expect(result.current.selectedIds.has('BBC One')).toBe(true)
    expect(result.current.allSelected).toBe(false)
  })

  it('should use custom getChannelKey function', () => {
    const customGetKey = (channel: Channel) => channel.tvg_id

    const { result } = renderHook(() => useChannelSelection(mockChannels, customGetKey))

    act(() => {
      result.current.handleSelectOne('bbc1.uk', true)
    })

    expect(result.current.selectedIds.has('bbc1.uk')).toBe(true)
    expect(result.current.selectedIds.has('BBC One')).toBe(false)
  })

  it('should handle selecting and deselecting multiple channels in sequence', () => {
    const { result } = renderHook(() => useChannelSelection(mockChannels, getChannelKey))

    act(() => {
      result.current.handleSelectOne('BBC One', true)
    })

    act(() => {
      result.current.handleSelectOne('ITV', true)
    })

    act(() => {
      result.current.handleSelectOne('TVE', true)
    })

    expect(result.current.selectedIds.size).toBe(3)
    expect(result.current.allSelected).toBe(true)

    act(() => {
      result.current.handleSelectOne('ITV', false)
    })

    expect(result.current.selectedIds.size).toBe(2)
    expect(result.current.allSelected).toBe(false)
    expect(result.current.someSelected).toBe(true)
  })
})
