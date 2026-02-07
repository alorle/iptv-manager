import { describe, it, expect, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useChannels } from './useChannels'
import * as channelsApi from '../api/channels'
import type { Channel } from '../types'

vi.mock('../api/channels')

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
]

describe('useChannels', () => {
  it('should initialize with loading state', () => {
    vi.spyOn(channelsApi, 'listChannels').mockImplementation(
      () => new Promise(() => {}) // Never resolves
    )

    const { result } = renderHook(() => useChannels())

    expect(result.current.loading).toBe(true)
    expect(result.current.channels).toEqual([])
    expect(result.current.error).toBeNull()
  })

  it('should provide refetch function', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const { result } = renderHook(() => useChannels())

    expect(typeof result.current.refetch).toBe('function')
  })

  it('should return channels array', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue([])

    const { result } = renderHook(() => useChannels())

    expect(Array.isArray(result.current.channels)).toBe(true)
  })

  it('should accept refreshTrigger parameter', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const { result } = renderHook(() => useChannels(1))

    expect(result.current).toBeDefined()
    expect(result.current.loading).toBeDefined()
  })

  it('should handle refreshTrigger changes', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const { result, rerender } = renderHook(({ trigger }) => useChannels(trigger), {
      initialProps: { trigger: 0 },
    })

    expect(result.current).toBeDefined()

    rerender({ trigger: 1 })

    expect(result.current).toBeDefined()
  })

  it('should maintain consistent return shape', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const { result } = renderHook(() => useChannels())

    expect(result.current).toHaveProperty('channels')
    expect(result.current).toHaveProperty('loading')
    expect(result.current).toHaveProperty('error')
    expect(result.current).toHaveProperty('refetch')
  })

  it('should initialize error as null', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const { result } = renderHook(() => useChannels())

    expect(result.current.error).toBeNull()
  })

  it('should initialize channels as empty array', () => {
    vi.spyOn(channelsApi, 'listChannels').mockImplementation(
      () => new Promise(() => {}) // Never resolves
    )

    const { result } = renderHook(() => useChannels())

    expect(result.current.channels).toEqual([])
  })

  it('should be able to re-render', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const { result, rerender } = renderHook(() => useChannels())

    expect(result.current).toBeDefined()

    rerender()

    expect(result.current).toBeDefined()
  })

  it('should handle different refreshTrigger values', () => {
    vi.spyOn(channelsApi, 'listChannels').mockResolvedValue(mockChannels)

    const triggers = [0, 1, 2, 3, 4]

    triggers.forEach((trigger) => {
      const { result } = renderHook(() => useChannels(trigger))
      expect(result.current).toBeDefined()
    })
  })
})
