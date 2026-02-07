import { describe, it, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useScrollDetection } from './useScrollDetection'

describe('useScrollDetection', () => {
  it('should initialize with isScrolled as false', () => {
    const { result } = renderHook(() => useScrollDetection())

    expect(result.current.isScrolled).toBe(false)
    expect(result.current.tableContainerRef.current).toBeNull()
  })

  it('should provide a ref object', () => {
    const { result } = renderHook(() => useScrollDetection())

    expect(result.current.tableContainerRef).toBeDefined()
    expect(typeof result.current.tableContainerRef).toBe('object')
  })

  it('should use default threshold of 20', () => {
    const { result } = renderHook(() => useScrollDetection())

    // The hook initializes with default threshold of 20
    // We can't directly test the threshold value but we can verify the hook initializes
    expect(result.current.isScrolled).toBe(false)
  })

  it('should accept custom threshold value', () => {
    const { result } = renderHook(() => useScrollDetection(50))

    // The hook accepts custom threshold
    expect(result.current.isScrolled).toBe(false)
  })

  it('should accept dependencies array', () => {
    const { result } = renderHook(() => useScrollDetection(20, [1, 2, 3]))

    expect(result.current.isScrolled).toBe(false)
  })

  it('should return stable ref object across rerenders', () => {
    const { result, rerender } = renderHook(() => useScrollDetection())

    const initialRef = result.current.tableContainerRef

    rerender()

    expect(result.current.tableContainerRef).toBe(initialRef)
  })

  it('should update when dependencies change', () => {
    const { result, rerender } = renderHook(({ deps }) => useScrollDetection(20, deps), {
      initialProps: { deps: [1] },
    })

    const initialRef = result.current.tableContainerRef

    rerender({ deps: [2] })

    // Ref should remain the same
    expect(result.current.tableContainerRef).toBe(initialRef)
  })

  it('should handle empty dependencies array', () => {
    const { result } = renderHook(() => useScrollDetection(20, []))

    expect(result.current.isScrolled).toBe(false)
    expect(result.current.tableContainerRef).toBeDefined()
  })

  it('should return object with correct shape', () => {
    const { result } = renderHook(() => useScrollDetection())

    expect(result.current).toHaveProperty('isScrolled')
    expect(result.current).toHaveProperty('tableContainerRef')
    expect(typeof result.current.isScrolled).toBe('boolean')
    expect(typeof result.current.tableContainerRef).toBe('object')
  })

  it('should handle multiple threshold values', () => {
    const thresholds = [0, 10, 20, 50, 100]

    thresholds.forEach((threshold) => {
      const { result } = renderHook(() => useScrollDetection(threshold))
      expect(result.current.isScrolled).toBe(false)
      expect(result.current.tableContainerRef).toBeDefined()
    })
  })
})
