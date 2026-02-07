import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Toast, type ToastMessage } from './Toast'

describe('Toast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  const mockToast: ToastMessage = {
    id: 'test-toast',
    type: 'info',
    message: 'Test message',
  }

  it('should render toast message', () => {
    const onClose = vi.fn()
    render(<Toast toast={mockToast} onClose={onClose} />)

    expect(screen.getByText('Test message')).toBeInTheDocument()
  })

  it('should render success icon for success type', () => {
    const onClose = vi.fn()
    const successToast: ToastMessage = { ...mockToast, type: 'success' }
    render(<Toast toast={successToast} onClose={onClose} />)

    const icon = screen.getByText('✓')
    expect(icon).toBeInTheDocument()
    expect(icon).toHaveAttribute('aria-hidden', 'true')
  })

  it('should render error icon for error type', () => {
    const onClose = vi.fn()
    const errorToast: ToastMessage = { ...mockToast, type: 'error' }
    render(<Toast toast={errorToast} onClose={onClose} />)

    expect(screen.getByText('✗')).toBeInTheDocument()
  })

  it('should render warning icon for warning type', () => {
    const onClose = vi.fn()
    const warningToast: ToastMessage = { ...mockToast, type: 'warning' }
    render(<Toast toast={warningToast} onClose={onClose} />)

    expect(screen.getByText('⚠')).toBeInTheDocument()
  })

  it('should render info icon for info type', () => {
    const onClose = vi.fn()
    const infoToast: ToastMessage = { ...mockToast, type: 'info' }
    render(<Toast toast={infoToast} onClose={onClose} />)

    expect(screen.getByText('ℹ')).toBeInTheDocument()
  })

  it('should call onClose when close button is clicked', async () => {
    vi.useRealTimers() // Temporarily use real timers for userEvent
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<Toast toast={mockToast} onClose={onClose} />)

    const closeButton = screen.getByRole('button', { name: /dismiss notification/i })
    await user.click(closeButton)

    expect(onClose).toHaveBeenCalledWith('test-toast')
    vi.useFakeTimers() // Restore fake timers for other tests
  })

  it('should auto-close after default duration (5000ms)', () => {
    const onClose = vi.fn()
    render(<Toast toast={mockToast} onClose={onClose} />)

    expect(onClose).not.toHaveBeenCalled()

    vi.advanceTimersByTime(5000)

    expect(onClose).toHaveBeenCalledWith('test-toast')
  })

  it('should auto-close after custom duration', () => {
    const onClose = vi.fn()
    const customToast: ToastMessage = { ...mockToast, duration: 3000 }
    render(<Toast toast={customToast} onClose={onClose} />)

    vi.advanceTimersByTime(2999)
    expect(onClose).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1)
    expect(onClose).toHaveBeenCalledWith('test-toast')
  })

  it('should have correct CSS class for type', () => {
    const onClose = vi.fn()
    const successToast: ToastMessage = { ...mockToast, type: 'success' }
    const { container } = render(<Toast toast={successToast} onClose={onClose} />)

    const toast = container.querySelector('.toast')
    expect(toast).toHaveClass('toast-success')
  })

  it('should have alert role for error type', () => {
    const onClose = vi.fn()
    const errorToast: ToastMessage = { ...mockToast, type: 'error' }
    const { container } = render(<Toast toast={errorToast} onClose={onClose} />)

    const toast = container.querySelector('.toast')
    expect(toast).toHaveAttribute('role', 'alert')
    expect(toast).toHaveAttribute('aria-live', 'assertive')
  })

  it('should have status role for non-error types', () => {
    const onClose = vi.fn()
    const infoToast: ToastMessage = { ...mockToast, type: 'info' }
    const { container } = render(<Toast toast={infoToast} onClose={onClose} />)

    const toast = container.querySelector('.toast')
    expect(toast).toHaveAttribute('role', 'status')
    expect(toast).toHaveAttribute('aria-live', 'polite')
  })

  it('should clear timeout on unmount', () => {
    const onClose = vi.fn()
    const { unmount } = render(<Toast toast={mockToast} onClose={onClose} />)

    unmount()

    vi.advanceTimersByTime(5000)

    expect(onClose).not.toHaveBeenCalled()
  })

  it('should reset timer when toast changes', () => {
    const onClose = vi.fn()
    const { rerender } = render(<Toast toast={mockToast} onClose={onClose} />)

    vi.advanceTimersByTime(3000)

    const newToast: ToastMessage = { ...mockToast, id: 'new-id', message: 'New message' }
    rerender(<Toast toast={newToast} onClose={onClose} />)

    vi.advanceTimersByTime(3000)
    expect(onClose).not.toHaveBeenCalled()

    vi.advanceTimersByTime(2000)
    expect(onClose).toHaveBeenCalledWith('new-id')
  })

  it('should render close button with correct attributes', () => {
    const onClose = vi.fn()
    render(<Toast toast={mockToast} onClose={onClose} />)

    const closeButton = screen.getByRole('button', { name: /dismiss notification/i })
    expect(closeButton).toHaveAttribute('type', 'button')
    expect(closeButton).toHaveTextContent('×')
  })

  it('should have toast-message class on message span', () => {
    const onClose = vi.fn()
    const { container } = render(<Toast toast={mockToast} onClose={onClose} />)

    const messageSpan = container.querySelector('.toast-message')
    expect(messageSpan).toBeInTheDocument()
    expect(messageSpan).toHaveTextContent('Test message')
  })

  it('should have toast-icon class on icon span', () => {
    const onClose = vi.fn()
    const { container } = render(<Toast toast={mockToast} onClose={onClose} />)

    const iconSpan = container.querySelector('.toast-icon')
    expect(iconSpan).toBeInTheDocument()
    expect(iconSpan).toHaveAttribute('aria-hidden', 'true')
  })

  it('should handle all toast types correctly', () => {
    const onClose = vi.fn()
    const types: Array<{ type: ToastMessage['type']; icon: string; role: string }> = [
      { type: 'success', icon: '✓', role: 'status' },
      { type: 'error', icon: '✗', role: 'alert' },
      { type: 'warning', icon: '⚠', role: 'status' },
      { type: 'info', icon: 'ℹ', role: 'status' },
    ]

    types.forEach(({ type, icon, role }) => {
      const toast: ToastMessage = { ...mockToast, type }
      const { container, unmount } = render(<Toast toast={toast} onClose={onClose} />)

      expect(screen.getByText(icon)).toBeInTheDocument()
      const toastElement = container.querySelector('.toast')
      expect(toastElement).toHaveAttribute('role', role)
      expect(toastElement).toHaveClass(`toast-${type}`)

      unmount()
    })
  })
})
