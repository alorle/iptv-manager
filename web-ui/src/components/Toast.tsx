import { useEffect } from 'react'
import './Toast.css'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface ToastMessage {
  id: string
  type: ToastType
  message: string
  duration?: number
}

interface ToastProps {
  toast: ToastMessage
  onClose: (id: string) => void
}

export function Toast({ toast, onClose }: ToastProps) {
  useEffect(() => {
    const duration = toast.duration ?? 5000
    const timer = setTimeout(() => {
      onClose(toast.id)
    }, duration)

    return () => clearTimeout(timer)
  }, [toast.id, toast.duration, onClose])

  const handleClose = () => {
    onClose(toast.id)
  }

  const getIcon = () => {
    switch (toast.type) {
      case 'success':
        return '✓'
      case 'error':
        return '✗'
      case 'warning':
        return '⚠'
      case 'info':
        return 'ℹ'
    }
  }

  const getAriaRole = () => {
    return toast.type === 'error' ? 'alert' : 'status'
  }

  return (
    <div
      className={`toast toast-${toast.type}`}
      role={getAriaRole()}
      aria-live={toast.type === 'error' ? 'assertive' : 'polite'}
    >
      <span className="toast-icon" aria-hidden="true">
        {getIcon()}
      </span>
      <span className="toast-message">{toast.message}</span>
      <button
        className="toast-close"
        onClick={handleClose}
        aria-label="Dismiss notification"
        type="button"
      >
        ×
      </button>
    </div>
  )
}
