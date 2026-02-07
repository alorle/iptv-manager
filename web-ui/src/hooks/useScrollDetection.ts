import { useState, useEffect, useRef } from 'react'

export interface ScrollDetectionResult {
  isScrolled: boolean
  tableContainerRef: React.RefObject<HTMLDivElement | null>
}

/**
 * Custom hook for detecting scroll position in a container
 * Used to apply visual effects like header shadows when scrolled
 */
export function useScrollDetection(
  threshold: number = 20,
  dependencies: unknown[] = []
): ScrollDetectionResult {
  const [isScrolled, setIsScrolled] = useState(false)
  const tableContainerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const tableContainer = tableContainerRef.current
    if (!tableContainer) return

    const handleScroll = () => {
      setIsScrolled(tableContainer.scrollTop > threshold)
    }

    // Check initial scroll position
    handleScroll()

    tableContainer.addEventListener('scroll', handleScroll, { passive: true })
    return () => tableContainer.removeEventListener('scroll', handleScroll)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, dependencies)

  return {
    isScrolled,
    tableContainerRef,
  }
}
