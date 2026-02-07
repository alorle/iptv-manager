import { useState } from 'react'
import { updateOverride, deleteOverride, type ValidationError } from '../api/channels'
import type { useToast } from './useToast'
import type { Channel } from '../types'
import type { CustomAttribute } from './useChannelOverrideForm'

export interface OverrideMutationsResult {
  saving: boolean
  validationError: string | null
  handleSave: (
    formData: {
      enabled: boolean
      tvgId: string
      tvgName: string
      tvgLogo: string
      groupTitle: string
      customAttributes: CustomAttribute[]
    },
    forceCheck: boolean
  ) => Promise<void>
  handleDelete: () => Promise<void>
  setValidationError: (error: string | null) => void
}

/**
 * Custom hook for channel override save and delete operations
 * Handles API calls, error handling, and toast notifications
 */
export function useChannelOverrideMutations(
  channel: Channel,
  toast: ReturnType<typeof useToast>,
  onSuccess: () => void,
  setTvgIdValidation: (validation: { valid: boolean; suggestions: string[] }) => void
): OverrideMutationsResult {
  const [saving, setSaving] = useState(false)
  const [validationError, setValidationError] = useState<string | null>(null)

  const stream = channel.streams[0]

  const handleSave = async (
    formData: {
      enabled: boolean
      tvgId: string
      tvgName: string
      tvgLogo: string
      groupTitle: string
      customAttributes: CustomAttribute[]
    },
    forceCheck: boolean
  ) => {
    setValidationError(null)
    setSaving(true)

    try {
      // Build override object with only changed fields
      const override: Record<string, unknown> = {}

      if (formData.enabled !== stream.enabled) {
        override.enabled = formData.enabled
      }

      if (formData.tvgId.trim() !== channel.tvg_id) {
        override.tvg_id = formData.tvgId.trim() || null
      }

      if (formData.tvgName.trim() !== stream.tvg_name) {
        override.tvg_name = formData.tvgName.trim() || null
      }

      if (formData.tvgLogo.trim() !== channel.tvg_logo) {
        override.tvg_logo = formData.tvgLogo.trim() || null
      }

      if (formData.groupTitle.trim() !== channel.group_title) {
        override.group_title = formData.groupTitle.trim() || null
      }

      // Add custom attributes (currently not supported by backend, placeholder)
      if (formData.customAttributes.length > 0) {
        const customAttrs: Record<string, string> = {}
        formData.customAttributes.forEach((attr) => {
          if (attr.key.trim()) {
            customAttrs[attr.key.trim()] = attr.value
          }
        })
        // TODO: Implement custom attributes support in backend
        // Custom attributes are collected but not yet persisted
        void customAttrs
      }

      await updateOverride(stream.acestream_id, override, forceCheck)
      toast.success('Channel override saved successfully')
      onSuccess()
    } catch (err) {
      if (err && typeof err === 'object' && 'error' in err) {
        const validationErr = err as ValidationError
        const errorMsg = validationErr.message || 'Validation failed'
        setValidationError(errorMsg)
        toast.error(errorMsg)
        if (validationErr.suggestions) {
          setTvgIdValidation({
            valid: false,
            suggestions: validationErr.suggestions,
          })
        }
      } else {
        const errorMsg = err instanceof Error ? err.message : 'Failed to save override'
        setValidationError(errorMsg)
        toast.error(errorMsg)
      }
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    setValidationError(null)
    setSaving(true)

    try {
      await deleteOverride(stream.acestream_id)
      toast.success('Channel override deleted successfully')
      onSuccess()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to delete override'
      setValidationError(errorMsg)
      toast.error(errorMsg)
    } finally {
      setSaving(false)
    }
  }

  return {
    saving,
    validationError,
    handleSave,
    handleDelete,
    setValidationError,
  }
}
