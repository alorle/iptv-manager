import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useToast } from './useToast'

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  it('should initialize with empty toasts array', () => {
    const { result } = renderHook(() => useToast())

    expect(result.current.toasts).toEqual([])
  })

  it('should add a success toast', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('Operation successful')
    })

    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].type).toBe('success')
    expect(result.current.toasts[0].message).toBe('Operation successful')
    expect(result.current.toasts[0].id).toBeDefined()
  })

  it('should add an error toast', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.error('Something went wrong')
    })

    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].type).toBe('error')
    expect(result.current.toasts[0].message).toBe('Something went wrong')
  })

  it('should add a warning toast', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.warning('Warning message')
    })

    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].type).toBe('warning')
    expect(result.current.toasts[0].message).toBe('Warning message')
  })

  it('should add an info toast', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.info('Information message')
    })

    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].type).toBe('info')
    expect(result.current.toasts[0].message).toBe('Information message')
  })

  it('should add multiple toasts', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('First message')
      result.current.error('Second message')
      result.current.warning('Third message')
    })

    expect(result.current.toasts).toHaveLength(3)
    expect(result.current.toasts[0].message).toBe('First message')
    expect(result.current.toasts[1].message).toBe('Second message')
    expect(result.current.toasts[2].message).toBe('Third message')
  })

  it('should close a specific toast by id', () => {
    const { result } = renderHook(() => useToast())

    let toastId: string

    act(() => {
      toastId = result.current.success('Toast to close')
    })

    expect(result.current.toasts).toHaveLength(1)

    act(() => {
      result.current.closeToast(toastId)
    })

    expect(result.current.toasts).toHaveLength(0)
  })

  it('should close only the specified toast when multiple exist', () => {
    const { result } = renderHook(() => useToast())

    let firstId: string
    let secondId: string

    act(() => {
      firstId = result.current.success('First toast')
      secondId = result.current.error('Second toast')
      result.current.warning('Third toast')
    })

    expect(result.current.toasts).toHaveLength(3)

    act(() => {
      result.current.closeToast(secondId)
    })

    expect(result.current.toasts).toHaveLength(2)
    expect(result.current.toasts.find((t) => t.id === firstId)).toBeDefined()
    expect(result.current.toasts.find((t) => t.id === secondId)).toBeUndefined()
  })

  it('should generate unique IDs for each toast', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('First')
      result.current.success('Second')
      result.current.success('Third')
    })

    const ids = result.current.toasts.map((t) => t.id)
    const uniqueIds = new Set(ids)

    expect(uniqueIds.size).toBe(3)
  })

  it('should support custom duration parameter', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('Custom duration', 5000)
    })

    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].duration).toBe(5000)
  })

  it('should use default duration when not specified', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('Default duration')
    })

    expect(result.current.toasts).toHaveLength(1)
    expect(result.current.toasts[0].duration).toBeUndefined()
  })

  it('should maintain toast order (FIFO)', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('First')
      result.current.error('Second')
      result.current.warning('Third')
    })

    expect(result.current.toasts[0].message).toBe('First')
    expect(result.current.toasts[1].message).toBe('Second')
    expect(result.current.toasts[2].message).toBe('Third')
  })

  it('should handle closing a non-existent toast gracefully', () => {
    const { result } = renderHook(() => useToast())

    act(() => {
      result.current.success('Test toast')
    })

    expect(result.current.toasts).toHaveLength(1)

    act(() => {
      result.current.closeToast('non-existent-id')
    })

    expect(result.current.toasts).toHaveLength(1)
  })

  it('should return toast ID from convenience methods', () => {
    const { result } = renderHook(() => useToast())

    let successId: string
    let errorId: string
    let warningId: string
    let infoId: string

    act(() => {
      successId = result.current.success('Success')
      errorId = result.current.error('Error')
      warningId = result.current.warning('Warning')
      infoId = result.current.info('Info')
    })

    expect(successId).toBeDefined()
    expect(errorId).toBeDefined()
    expect(warningId).toBeDefined()
    expect(infoId).toBeDefined()
    expect(result.current.toasts.find((t) => t.id === successId)).toBeDefined()
    expect(result.current.toasts.find((t) => t.id === errorId)).toBeDefined()
    expect(result.current.toasts.find((t) => t.id === warningId)).toBeDefined()
    expect(result.current.toasts.find((t) => t.id === infoId)).toBeDefined()
  })
})
