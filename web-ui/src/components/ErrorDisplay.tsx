import './ErrorDisplay.css'

interface ErrorDisplayProps {
  error: Error
  onRetry?: () => void
  title?: string
}

export function ErrorDisplay({ error, onRetry, title = 'Error' }: ErrorDisplayProps) {
  const isNetworkError =
    error.message.includes('Failed to fetch') ||
    error.message.includes('NetworkError') ||
    error.message.includes('network')

  return (
    <div className="error-display">
      <div className="error-icon">⚠</div>
      <h3 className="error-title">{title}</h3>
      <p className="error-message">{error.message}</p>
      {isNetworkError && (
        <p className="error-hint">
          The API server may be unreachable. Please check your connection.
        </p>
      )}
      {onRetry && (
        <button className="error-retry-button" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  )
}
