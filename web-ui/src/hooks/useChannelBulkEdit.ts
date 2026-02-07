import { bulkUpdateOverrides } from '../api/channels'
import type { useToast } from './useToast'

export interface BulkEditResult {
  handleBulkEdit: (
    selectedIds: Set<string>,
    field: string,
    value: string | boolean
  ) => Promise<void>
}

/**
 * Custom hook for bulk editing channel overrides
 * Handles API calls and error handling for bulk operations
 */
export function useChannelBulkEdit(
  toast: ReturnType<typeof useToast>,
  onSuccess: () => void
): BulkEditResult {
  const handleBulkEdit = async (
    selectedIds: Set<string>,
    field: string,
    value: string | boolean
  ) => {
    try {
      const result = await bulkUpdateOverrides(Array.from(selectedIds), field, value)

      if (result.failed > 0) {
        toast.warning(`Updated ${result.updated} channel(s), but ${result.failed} failed`, 7000)
      } else {
        toast.success(`Successfully updated ${result.updated} channel(s)`)
      }

      onSuccess()
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update channels'
      toast.error(errorMessage)
      throw err
    }
  }

  return {
    handleBulkEdit,
  }
}
