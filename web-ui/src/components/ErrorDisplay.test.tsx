import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ErrorDisplay } from './ErrorDisplay'

describe('ErrorDisplay', () => {
  it('should render error message', () => {
    const error = new Error('Something went wrong')
    render(<ErrorDisplay error={error} />)

    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('should render default title', () => {
    const error = new Error('Test error')
    render(<ErrorDisplay error={error} />)

    expect(screen.getByText('Error')).toBeInTheDocument()
  })

  it('should render custom title', () => {
    const error = new Error('Test error')
    render(<ErrorDisplay error={error} title="Custom Error" />)

    expect(screen.getByText('Custom Error')).toBeInTheDocument()
  })

  it('should render error icon', () => {
    const error = new Error('Test error')
    const { container } = render(<ErrorDisplay error={error} />)

    const icon = container.querySelector('.error-icon')
    expect(icon).toBeInTheDocument()
    expect(icon).toHaveTextContent('⚠')
  })

  it('should render retry button when onRetry is provided', () => {
    const error = new Error('Test error')
    const onRetry = vi.fn()
    render(<ErrorDisplay error={error} onRetry={onRetry} />)

    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
  })

  it('should not render retry button when onRetry is not provided', () => {
    const error = new Error('Test error')
    render(<ErrorDisplay error={error} />)

    expect(screen.queryByRole('button', { name: /retry/i })).not.toBeInTheDocument()
  })

  it('should call onRetry when retry button is clicked', async () => {
    const user = userEvent.setup()
    const error = new Error('Test error')
    const onRetry = vi.fn()
    render(<ErrorDisplay error={error} onRetry={onRetry} />)

    const retryButton = screen.getByRole('button', { name: /retry/i })
    await user.click(retryButton)

    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('should show network error hint for "Failed to fetch" errors', () => {
    const error = new Error('Failed to fetch')
    render(<ErrorDisplay error={error} />)

    expect(screen.getByText(/the api server may be unreachable/i)).toBeInTheDocument()
  })

  it('should show network error hint for "NetworkError" errors', () => {
    const error = new Error('NetworkError when attempting to fetch resource')
    render(<ErrorDisplay error={error} />)

    expect(screen.getByText(/the api server may be unreachable/i)).toBeInTheDocument()
  })

  it('should show network error hint for errors containing "network"', () => {
    const error = new Error('A network error occurred')
    render(<ErrorDisplay error={error} />)

    expect(screen.getByText(/the api server may be unreachable/i)).toBeInTheDocument()
  })

  it('should not show network error hint for non-network errors', () => {
    const error = new Error('Invalid input')
    render(<ErrorDisplay error={error} />)

    expect(screen.queryByText(/the api server may be unreachable/i)).not.toBeInTheDocument()
  })

  it('should handle all props together', async () => {
    const user = userEvent.setup()
    const error = new Error('Failed to fetch data')
    const onRetry = vi.fn()
    render(<ErrorDisplay error={error} onRetry={onRetry} title="Network Error" />)

    expect(screen.getByText('Network Error')).toBeInTheDocument()
    expect(screen.getByText('Failed to fetch data')).toBeInTheDocument()
    expect(screen.getByText(/the api server may be unreachable/i)).toBeInTheDocument()

    const retryButton = screen.getByRole('button', { name: /retry/i })
    await user.click(retryButton)

    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('should have correct CSS classes', () => {
    const error = new Error('Test error')
    const { container } = render(<ErrorDisplay error={error} />)

    expect(container.querySelector('.error-display')).toBeInTheDocument()
    expect(container.querySelector('.error-icon')).toBeInTheDocument()
    expect(container.querySelector('.error-title')).toBeInTheDocument()
    expect(container.querySelector('.error-message')).toBeInTheDocument()
  })

  it('should render network hint with correct CSS class', () => {
    const error = new Error('Failed to fetch')
    const { container } = render(<ErrorDisplay error={error} />)

    const hint = container.querySelector('.error-hint')
    expect(hint).toBeInTheDocument()
    expect(hint).toHaveTextContent(/the api server may be unreachable/i)
  })

  it('should render retry button with correct CSS class', () => {
    const error = new Error('Test error')
    const onRetry = vi.fn()
    const { container } = render(<ErrorDisplay error={error} onRetry={onRetry} />)

    const button = container.querySelector('.error-retry-button')
    expect(button).toBeInTheDocument()
  })
})
