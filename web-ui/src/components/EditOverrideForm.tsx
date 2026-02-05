import { useState, useEffect, useRef, useCallback } from 'react'
import type { Channel } from '../types'
import {
  updateOverride,
  deleteOverride,
  validateTvgId,
  type ValidationError,
} from '../api/channels'
import { ConfirmDialog } from './ConfirmDialog'
import { LoadingSpinner } from './LoadingSpinner'
import type { useToast } from '../hooks/useToast'
import './EditOverrideForm.css'

interface EditOverrideFormProps {
  channel: Channel
  onClose: () => void
  onSave: () => void
  toast: ReturnType<typeof useToast>
}

interface CustomAttribute {
  key: string
  value: string
}

export function EditOverrideForm({ channel, onClose, onSave, toast }: EditOverrideFormProps) {
  // Extract the first (and only) stream from the channel
  const stream = channel.streams[0]

  const [enabled, setEnabled] = useState<boolean>(stream.enabled)
  const [tvgId, setTvgId] = useState<string>(channel.tvg_id)
  const [tvgName, setTvgName] = useState<string>(stream.tvg_name)
  const [tvgLogo, setTvgLogo] = useState<string>(channel.tvg_logo)
  const [groupTitle, setGroupTitle] = useState<string>(channel.group_title)
  const [customAttributes, setCustomAttributes] = useState<CustomAttribute[]>([])

  const [tvgIdValidation, setTvgIdValidation] = useState<{
    valid: boolean
    suggestions: string[]
  } | null>(null)
  const [validating, setValidating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [forceCheck, setForceCheck] = useState(false)
  const [validationError, setValidationError] = useState<string | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const modalRef = useRef<HTMLDivElement>(null)
  const closeButtonRef = useRef<HTMLButtonElement>(null)

  // Focus management - focus close button when modal opens
  useEffect(() => {
    closeButtonRef.current?.focus()

    // Store previously focused element to restore later
    const previouslyFocusedElement = document.activeElement as HTMLElement

    return () => {
      previouslyFocusedElement?.focus()
    }
  }, [])

  // Handle escape key to close modal
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Escape' && !showDeleteConfirm) {
        onClose()
      }
    },
    [onClose, showDeleteConfirm]
  )

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
      if (trimmedTvgId === channel.tvg_id) {
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
    [channel.tvg_id]
  )

  // Debounced TVG-ID validation while typing
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      performValidation(tvgId)
    }, 300) // 300ms debounce

    return () => clearTimeout(timeoutId)
  }, [tvgId, performValidation])

  // Handle blur event to validate immediately
  const handleTvgIdBlur = () => {
    performValidation(tvgId)
  }

  const handleAddCustomAttribute = () => {
    setCustomAttributes([...customAttributes, { key: '', value: '' }])
  }

  const handleRemoveCustomAttribute = (index: number) => {
    setCustomAttributes(customAttributes.filter((_, i) => i !== index))
  }

  const handleCustomAttributeChange = (index: number, field: 'key' | 'value', value: string) => {
    const updated = [...customAttributes]
    updated[index][field] = value
    setCustomAttributes(updated)
  }

  const handleSave = async () => {
    setValidationError(null)
    setSaving(true)

    try {
      // Build override object with only changed fields
      const override: Record<string, unknown> = {}

      if (enabled !== stream.enabled) {
        override.enabled = enabled
      }

      if (tvgId.trim() !== channel.tvg_id) {
        override.tvg_id = tvgId.trim() || null
      }

      if (tvgName.trim() !== stream.tvg_name) {
        override.tvg_name = tvgName.trim() || null
      }

      if (tvgLogo.trim() !== channel.tvg_logo) {
        override.tvg_logo = tvgLogo.trim() || null
      }

      if (groupTitle.trim() !== channel.group_title) {
        override.group_title = groupTitle.trim() || null
      }

      // Add custom attributes (currently not supported by backend, placeholder)
      if (customAttributes.length > 0) {
        const customAttrs: Record<string, string> = {}
        customAttributes.forEach((attr) => {
          if (attr.key.trim()) {
            customAttrs[attr.key.trim()] = attr.value
          }
        })
        if (Object.keys(customAttrs).length > 0) {
          // Store as a comment for now since backend doesn't support it yet
          console.log('Custom attributes not yet supported:', customAttrs)
        }
      }

      await updateOverride(stream.acestream_id, override, forceCheck)
      toast.success('Channel override saved successfully')
      onSave()
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

  const handleDeleteConfirm = async () => {
    setShowDeleteConfirm(false)
    setValidationError(null)
    setSaving(true)

    try {
      await deleteOverride(stream.acestream_id)
      toast.success('Channel override deleted successfully')
      onSave()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to delete override'
      setValidationError(errorMsg)
      toast.error(errorMsg)
    } finally {
      setSaving(false)
    }
  }

  const handleSuggestionClick = (suggestion: string) => {
    setTvgId(suggestion)
  }

  const isTvgIdInvalid = tvgIdValidation && !tvgIdValidation.valid && tvgId.trim() !== ''
  const canSave = !validating && (!isTvgIdInvalid || forceCheck)

  return (
    <div
      className="edit-override-form-overlay"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="edit-form-title"
      onKeyDown={handleKeyDown}
    >
      <div ref={modalRef} className="edit-override-form" onClick={(e) => e.stopPropagation()}>
        <div className="form-header">
          <h2 id="edit-form-title">Edit Channel Override</h2>
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

        <div className="form-content">
          <div className="channel-info">
            <h3>{stream.name}</h3>
            <p className="acestream-id">ID: {stream.acestream_id}</p>
          </div>

          {validationError && <div className="error-message">{validationError}</div>}

          <div className="form-field">
            <label>
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              <span className="toggle-label">Channel Enabled</span>
            </label>
          </div>

          <div className="form-field">
            <label htmlFor="tvg-id">TVG-ID</label>
            <input
              id="tvg-id"
              type="text"
              value={tvgId}
              onChange={(e) => setTvgId(e.target.value)}
              onBlur={handleTvgIdBlur}
              placeholder={channel.tvg_id || 'Original TVG-ID'}
            />
            {validating && <span className="validation-status validating">Validating...</span>}
            {!validating && tvgIdValidation && tvgId.trim() !== '' && (
              <span className={`validation-status ${tvgIdValidation.valid ? 'valid' : 'invalid'}`}>
                {tvgIdValidation.valid ? '✓ Valid' : '✗ Invalid'}
              </span>
            )}
            {isTvgIdInvalid && tvgIdValidation.suggestions.length > 0 && (
              <div className="suggestions">
                <p className="suggestions-label">Did you mean:</p>
                <ul>
                  {tvgIdValidation.suggestions.map((suggestion) => (
                    <li key={suggestion} onClick={() => handleSuggestionClick(suggestion)}>
                      {suggestion}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          <div className="form-field">
            <label htmlFor="tvg-name">TVG-Name</label>
            <input
              id="tvg-name"
              type="text"
              value={tvgName}
              onChange={(e) => setTvgName(e.target.value)}
              placeholder={stream.tvg_name || 'Original TVG-Name'}
            />
          </div>

          <div className="form-field">
            <label htmlFor="tvg-logo">TVG-Logo URL</label>
            <input
              id="tvg-logo"
              type="text"
              value={tvgLogo}
              onChange={(e) => setTvgLogo(e.target.value)}
              placeholder={channel.tvg_logo || 'Original TVG-Logo'}
            />
          </div>

          <div className="form-field">
            <label htmlFor="group-title">Group Title</label>
            <input
              id="group-title"
              type="text"
              value={groupTitle}
              onChange={(e) => setGroupTitle(e.target.value)}
              placeholder={channel.group_title || 'Original Group Title'}
            />
          </div>

          <div className="custom-attributes-section">
            <div className="section-header">
              <h4>Custom Attributes</h4>
              <button
                type="button"
                className="add-attribute-button"
                onClick={handleAddCustomAttribute}
              >
                + Add
              </button>
            </div>

            {customAttributes.map((attr, index) => (
              <div key={index} className="custom-attribute">
                <input
                  type="text"
                  placeholder="Key"
                  value={attr.key}
                  onChange={(e) => handleCustomAttributeChange(index, 'key', e.target.value)}
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={attr.value}
                  onChange={(e) => handleCustomAttributeChange(index, 'value', e.target.value)}
                />
                <button
                  type="button"
                  className="remove-attribute-button"
                  onClick={() => handleRemoveCustomAttribute(index)}
                >
                  Remove
                </button>
              </div>
            ))}

            {customAttributes.length === 0 && (
              <p className="no-attributes">No custom attributes. Click "Add" to create one.</p>
            )}
          </div>

          {isTvgIdInvalid && (
            <div className="form-field">
              <label>
                <input
                  type="checkbox"
                  checked={forceCheck}
                  onChange={(e) => setForceCheck(e.target.checked)}
                />
                <span className="force-label">Force save (skip validation)</span>
              </label>
            </div>
          )}
        </div>

        <div className="form-actions">
          <button className="save-button" onClick={handleSave} disabled={saving || !canSave}>
            {saving ? (
              <>
                Saving
                <LoadingSpinner size="small" inline />
              </>
            ) : (
              'Save'
            )}
          </button>
          <button className="cancel-button" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          {stream.has_override && (
            <button
              className="delete-button"
              onClick={() => setShowDeleteConfirm(true)}
              disabled={saving}
            >
              Delete Override
            </button>
          )}
        </div>
      </div>

      {showDeleteConfirm && (
        <ConfirmDialog
          title="Delete Channel Override"
          message={`Are you sure you want to delete the override for "${stream.name}"? This action cannot be undone.`}
          confirmText="Delete"
          cancelText="Cancel"
          confirmVariant="danger"
          onConfirm={handleDeleteConfirm}
          onCancel={() => setShowDeleteConfirm(false)}
        />
      )}
    </div>
  )
}
