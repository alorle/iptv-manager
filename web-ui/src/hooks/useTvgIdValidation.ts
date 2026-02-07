import { useState, useEffect, useCallback } from 'react'
import { validateTvgId } from '../api/channels'

export interface TvgIdValidationResult {
  tvgIdValidation: {
    valid: boolean
    suggestions: string[]
  } | null
  validating: boolean
  isTvgIdInvalid: boolean
  performValidation: (value: string) => Promise<void>
  handleTvgIdBlur: () => void
  handleSuggestionClick: (suggestion: string) => void
  setTvgIdValidation: (validation: { valid: boolean; suggestions: string[] } | null) => void
}

/**
 * Custom hook for TVG-ID validation with debouncing
 * Handles validation API calls and suggestion management
 */
export function useTvgIdValidation(
  tvgId: string,
  originalTvgId: string,
  setTvgId: (value: string) => void
): TvgIdValidationResult {
  const [tvgIdValidation, setTvgIdValidation] = useState<{
    valid: boolean
    suggestions: string[]
  } | null>(null)
  const [validating, setValidating] = useState(false)

  // Validation function
  const performValidation = useCallback(
    async (value: string) => {
      const trimmedTvgId = value.trim()

      // Empty TVG-ID is always valid
      if (trimmedTvgId === '') {
        setTvgIdValidation({ valid: true, suggestions: [] })
        return
      }

      // Don't validate if same as original
      if (trimmedTvgId === originalTvgId) {
        setTvgIdValidation({ valid: true, suggestions: [] })
        return
      }

      setValidating(true)
      try {
        const result = await validateTvgId(trimmedTvgId)
        setTvgIdValidation({
          valid: result.valid,
          suggestions: result.suggestions || [],
        })
      } catch (err) {
        console.error('Failed to validate TVG-ID:', err)
        setTvgIdValidation({ valid: true, suggestions: [] }) // Assume valid on error
      } finally {
        setValidating(false)
      }
    },
    [originalTvgId]
  )

  // Debounced TVG-ID validation while typing
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      performValidation(tvgId)
    }, 300) // 300ms debounce

    return () => clearTimeout(timeoutId)
  }, [tvgId, performValidation])

  // Handle blur event to validate immediately
  const handleTvgIdBlur = useCallback(() => {
    performValidation(tvgId)
  }, [tvgId, performValidation])

  // Handle clicking on a suggestion
  const handleSuggestionClick = useCallback(
    (suggestion: string) => {
      setTvgId(suggestion)
    },
    [setTvgId]
  )

  const isTvgIdInvalid = Boolean(
    tvgIdValidation && !tvgIdValidation.valid && tvgId.trim() !== ''
  )

  return {
    tvgIdValidation,
    validating,
    isTvgIdInvalid,
    performValidation,
    handleTvgIdBlur,
    handleSuggestionClick,
    setTvgIdValidation,
  }
}
