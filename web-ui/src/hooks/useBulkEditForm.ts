import { useState } from 'react'

export type BulkEditField = 'enabled' | 'tvg_id' | 'tvg_name' | 'tvg_logo' | 'group_title'

export interface BulkEditFormState {
  field: BulkEditField
  value: string
  submitting: boolean
  error: string | null
  handleFieldChange: (newField: BulkEditField) => void
  setValue: (value: string) => void
  handleSubmit: (
    e: React.FormEvent,
    onSubmit: (field: string, value: string | boolean) => Promise<void>,
    onClose: () => void
  ) => Promise<void>
}

/**
 * Custom hook for bulk edit form state and logic
 * Handles field selection, value management, and form submission
 */
export function useBulkEditForm(): BulkEditFormState {
  const [field, setField] = useState<BulkEditField>('enabled')
  const [value, setValue] = useState<string>('true')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Reset value when field changes
  const handleFieldChange = (newField: BulkEditField) => {
    setField(newField)
    // Set default value based on field type
    setValue(newField === 'enabled' ? 'true' : '')
  }

  const handleSubmit = async (
    e: React.FormEvent,
    onSubmit: (field: string, value: string | boolean) => Promise<void>,
    onClose: () => void
  ) => {
    e.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      let processedValue: string | boolean = value

      // Convert enabled field to boolean
      if (field === 'enabled') {
        processedValue = value === 'true'
      }

      await onSubmit(field, processedValue)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update channels')
    } finally {
      setSubmitting(false)
    }
  }

  return {
    field,
    value,
    submitting,
    error,
    handleFieldChange,
    setValue,
    handleSubmit,
  }
}
