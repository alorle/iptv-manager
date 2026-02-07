import { useEffect, useRef } from 'react'

export interface FocusManagementResult {
  closeButtonRef: React.RefObject<HTMLButtonElement | null>
}

/**
 * Custom hook for managing focus in modal dialogs
 * Focuses the close button on mount and restores focus on unmount
 */
export function useFocusManagement(): FocusManagementResult {
  const closeButtonRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    closeButtonRef.current?.focus()

    // Store previously focused element to restore later
    const previouslyFocusedElement = document.activeElement as HTMLElement

    return () => {
      previouslyFocusedElement?.focus()
    }
  }, [])

  return {
    closeButtonRef,
  }
}
