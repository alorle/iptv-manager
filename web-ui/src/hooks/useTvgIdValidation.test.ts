import { describe, it, expect, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useTvgIdValidation } from './useTvgIdValidation'

describe('useTvgIdValidation', () => {
  it('should initialize with null validation state', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('', '', setTvgId))

    expect(result.current.tvgIdValidation).toBeNull()
    expect(result.current.validating).toBe(false)
    expect(result.current.isTvgIdInvalid).toBe(false)
  })

  it('should provide validation functions', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('test', 'original', setTvgId))

    expect(typeof result.current.performValidation).toBe('function')
    expect(typeof result.current.handleTvgIdBlur).toBe('function')
    expect(typeof result.current.handleSuggestionClick).toBe('function')
    expect(typeof result.current.setTvgIdValidation).toBe('function')
  })

  it('should handle suggestion click by setting TVG-ID', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('test.id', 'original.id', setTvgId))

    result.current.handleSuggestionClick('suggested.id')

    expect(setTvgId).toHaveBeenCalledWith('suggested.id')
  })

  it('should allow manual setting of validation state', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('test.id', 'original.id', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: false, suggestions: ['manual.suggestion'] })
    })

    expect(result.current.tvgIdValidation).toEqual({
      valid: false,
      suggestions: ['manual.suggestion'],
    })
  })

  it('should not show invalid state when validation is null', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('test', 'original', setTvgId))

    expect(result.current.isTvgIdInvalid).toBe(false)
  })

  it('should show invalid state when validation fails and TVG-ID is not empty', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('invalid.id', 'original.id', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: false, suggestions: [] })
    })

    expect(result.current.isTvgIdInvalid).toBe(true)
  })

  it('should not show invalid state when validation fails but TVG-ID is empty', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('', 'original.id', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: false, suggestions: [] })
    })

    expect(result.current.isTvgIdInvalid).toBe(false)
  })

  it('should not show invalid state when validation succeeds', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('valid.id', 'original.id', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: true, suggestions: [] })
    })

    expect(result.current.isTvgIdInvalid).toBe(false)
  })

  it('should handle different tvgId values', () => {
    const setTvgId = vi.fn()
    const { result, rerender } = renderHook(
      ({ tvgId }) => useTvgIdValidation(tvgId, 'original.id', setTvgId),
      { initialProps: { tvgId: 'test1' } }
    )

    expect(result.current.tvgIdValidation).toBeNull()

    rerender({ tvgId: 'test2' })

    // Hook should handle the change
    expect(result.current).toBeDefined()
  })

  it('should handle different originalTvgId values', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('test', 'original', setTvgId))

    expect(result.current).toBeDefined()
    expect(result.current.tvgIdValidation).toBeNull()
  })

  it('should maintain validation state across rerenders', () => {
    const setTvgId = vi.fn()
    const { result, rerender } = renderHook(() => useTvgIdValidation('test', 'original', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: true, suggestions: ['sug1'] })
    })

    expect(result.current.tvgIdValidation).toEqual({ valid: true, suggestions: ['sug1'] })

    rerender()

    expect(result.current.tvgIdValidation).toEqual({ valid: true, suggestions: ['sug1'] })
  })

  it('should handle whitespace in TVG-ID for isTvgIdInvalid calculation', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('   ', 'original', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: false, suggestions: [] })
    })

    // Whitespace-only string should be treated as empty
    expect(result.current.isTvgIdInvalid).toBe(false)
  })

  it('should properly calculate isTvgIdInvalid with trimmed empty string', () => {
    const setTvgId = vi.fn()
    const { result } = renderHook(() => useTvgIdValidation('', 'original', setTvgId))

    act(() => {
      result.current.setTvgIdValidation({ valid: false, suggestions: [] })
    })

    expect(result.current.isTvgIdInvalid).toBe(false)
  })

  it('should return stable function references across renders', () => {
    const setTvgId = vi.fn()
    const { result, rerender } = renderHook(() => useTvgIdValidation('test', 'original', setTvgId))

    const {
      performValidation,
      handleTvgIdBlur,
      handleSuggestionClick,
      setTvgIdValidation: setValidation,
    } = result.current

    rerender()

    expect(result.current.performValidation).toBe(performValidation)
    expect(result.current.handleTvgIdBlur).toBe(handleTvgIdBlur)
    expect(result.current.handleSuggestionClick).toBe(handleSuggestionClick)
    expect(result.current.setTvgIdValidation).toBe(setValidation)
  })
})
