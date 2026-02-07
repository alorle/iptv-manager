import { useState, useCallback, useRef } from 'react'
import type { Channel } from '../types'
import { useChannelOverrideForm } from '../hooks/useChannelOverrideForm'
import { useTvgIdValidation } from '../hooks/useTvgIdValidation'
import { useChannelOverrideMutations } from '../hooks/useChannelOverrideMutations'
import { useFocusManagement } from '../hooks/useFocusManagement'
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

export function EditOverrideForm({ channel, onClose, onSave, toast }: EditOverrideFormProps) {
  const stream = channel.streams[0]
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const modalRef = useRef<HTMLDivElement>(null)

  // Custom hooks for logic separation
  const {
    enabled,
    tvgId,
    tvgName,
    tvgLogo,
    groupTitle,
    customAttributes,
    forceCheck,
    setEnabled,
    setTvgId,
    setTvgName,
    setTvgLogo,
    setGroupTitle,
    setForceCheck,
    handleAddCustomAttribute,
    handleRemoveCustomAttribute,
    handleCustomAttributeChange,
  } = useChannelOverrideForm(channel)

  const {
    tvgIdValidation,
    validating,
    isTvgIdInvalid,
    handleTvgIdBlur,
    handleSuggestionClick,
    setTvgIdValidation,
  } = useTvgIdValidation(tvgId, channel.tvg_id, setTvgId)

  const { closeButtonRef } = useFocusManagement()

  const {
    saving,
    validationError,
    handleSave: saveOperation,
    handleDelete,
  } = useChannelOverrideMutations(channel, toast, onSave, setTvgIdValidation)

  // Wrapper to pass form data to save operation
  const handleSave = async () => {
    await saveOperation(
      {
        enabled,
        tvgId,
        tvgName,
        tvgLogo,
        groupTitle,
        customAttributes,
      },
      forceCheck
    )
  }

  // Handle delete confirmation
  const handleDeleteConfirm = async () => {
    setShowDeleteConfirm(false)
    await handleDelete()
  }

  // Handle escape key to close modal
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Escape' && !showDeleteConfirm) {
        onClose()
      }
    },
    [onClose, showDeleteConfirm]
  )

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
            {isTvgIdInvalid && tvgIdValidation && tvgIdValidation.suggestions.length > 0 && (
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
