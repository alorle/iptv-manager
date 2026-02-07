import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useChannelFilters } from './useChannelFilters'
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
        enabled: false,
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

describe('useChannelFilters', () => {
  it('should initialize with default filter values', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    expect(result.current.searchText).toBe('')
    expect(result.current.groupFilter).toBe('')
    expect(result.current.enabledFilter).toBe('enabled')
    expect(result.current.filteredChannels).toHaveLength(2) // Only enabled by default
  })

  it('should return all channels when no filters are applied and enabledFilter is "all"', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('all')
    })

    expect(result.current.filteredChannels).toHaveLength(3)
  })

  it('should filter channels by search text (case insensitive)', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('all')
      result.current.setSearchText('bbc')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
    expect(result.current.filteredChannels[0].name).toBe('BBC One')
  })

  it('should filter channels by search text with uppercase', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('all')
      result.current.setSearchText('BBC')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
    expect(result.current.filteredChannels[0].name).toBe('BBC One')
  })

  it('should filter channels by group', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('all')
      result.current.setGroupFilter('Spain')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
    expect(result.current.filteredChannels[0].name).toBe('TVE')
  })

  it('should filter channels by enabled status (enabled only)', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('enabled')
    })

    expect(result.current.filteredChannels).toHaveLength(2)
    expect(result.current.filteredChannels.every((ch) => ch.streams.some((s) => s.enabled))).toBe(
      true
    )
  })

  it('should filter channels by enabled status (disabled only)', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('disabled')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
    expect(result.current.filteredChannels[0].name).toBe('ITV')
  })

  it('should combine multiple filters', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('all')
      result.current.setGroupFilter('UK')
      result.current.setSearchText('bbc')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
    expect(result.current.filteredChannels[0].name).toBe('BBC One')
  })

  it('should return unique groups sorted alphabetically', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    expect(result.current.uniqueGroups).toEqual(['Spain', 'UK'])
  })

  it('should update uniqueGroups when channels change', () => {
    const { result, rerender } = renderHook(({ channels }) => useChannelFilters(channels), {
      initialProps: { channels: mockChannels },
    })

    expect(result.current.uniqueGroups).toEqual(['Spain', 'UK'])

    const newChannels: Channel[] = [
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

    expect(result.current.uniqueGroups).toEqual(['France'])
  })

  it('should return empty array when no channels match filters', () => {
    const { result } = renderHook(() => useChannelFilters(mockChannels))

    act(() => {
      result.current.setEnabledFilter('all')
      result.current.setSearchText('nonexistent')
    })

    expect(result.current.filteredChannels).toHaveLength(0)
  })

  it('should handle channels with no group title', () => {
    const channelsWithNoGroup: Channel[] = [
      {
        name: 'Test Channel',
        tvg_id: 'test.com',
        tvg_logo: '',
        group_title: '',
        streams: [
          {
            acestream_id: 'test123',
            name: 'Test',
            tvg_name: 'Test',
            source: 'elcano',
            enabled: true,
            has_override: false,
          },
        ],
      },
    ]

    const { result } = renderHook(() => useChannelFilters(channelsWithNoGroup))

    expect(result.current.uniqueGroups).toEqual([])
    expect(result.current.filteredChannels).toHaveLength(1)
  })

  it('should consider channel enabled if at least one stream is enabled', () => {
    const channelsWithMultipleStreams: Channel[] = [
      {
        name: 'Multi Stream',
        tvg_id: 'multi.com',
        tvg_logo: '',
        group_title: 'Test',
        streams: [
          {
            acestream_id: 'stream1',
            name: 'Stream 1',
            tvg_name: 'Multi',
            source: 'elcano',
            enabled: false,
            has_override: false,
          },
          {
            acestream_id: 'stream2',
            name: 'Stream 2',
            tvg_name: 'Multi',
            source: 'newera',
            enabled: true,
            has_override: false,
          },
        ],
      },
    ]

    const { result } = renderHook(() => useChannelFilters(channelsWithMultipleStreams))

    act(() => {
      result.current.setEnabledFilter('enabled')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
  })

  it('should consider channel disabled if all streams are disabled', () => {
    const channelsWithAllDisabled: Channel[] = [
      {
        name: 'All Disabled',
        tvg_id: 'disabled.com',
        tvg_logo: '',
        group_title: 'Test',
        streams: [
          {
            acestream_id: 'stream1',
            name: 'Stream 1',
            tvg_name: 'Disabled',
            source: 'elcano',
            enabled: false,
            has_override: false,
          },
          {
            acestream_id: 'stream2',
            name: 'Stream 2',
            tvg_name: 'Disabled',
            source: 'newera',
            enabled: false,
            has_override: false,
          },
        ],
      },
    ]

    const { result } = renderHook(() => useChannelFilters(channelsWithAllDisabled))

    act(() => {
      result.current.setEnabledFilter('disabled')
    })

    expect(result.current.filteredChannels).toHaveLength(1)
  })
})
