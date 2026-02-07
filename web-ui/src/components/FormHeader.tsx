import { RefObject } from 'react'

interface FormHeaderProps {
  title: string
  onClose: () => void
  closeButtonRef: RefObject<HTMLButtonElement>
}

export function FormHeader({ title, onClose, closeButtonRef }: FormHeaderProps) {
  return (
    <div className="form-header">
      <h2 id="edit-form-title">{title}</h2>
      <button
        ref={closeButtonRef}
        className="close-button"
        onClick={onClose}
        aria-label="Close dialog"
        type="button"
      >
        ×
      </button>
    </div>
  )
}
