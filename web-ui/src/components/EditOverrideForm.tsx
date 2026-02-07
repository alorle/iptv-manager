import { useState, useCallback, useRef } from 'react'
import type { Channel } from '../types'
import { useChannelOverrideForm } from '../hooks/useChannelOverrideForm'
import { useTvgIdValidation } from '../hooks/useTvgIdValidation'
import { useChannelOverrideMutations } from '../hooks/useChannelOverrideMutations'
import { useFocusManagement } from '../hooks/useFocusManagement'
import { ConfirmDialog } from './ConfirmDialog'
import { LoadingSpinner } from './LoadingSpinner'
import { FormField } from './FormField'
import { TvgIdField } from './TvgIdField'
import { CustomAttributesSection } from './CustomAttributesSection'
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

          <TvgIdField
            tvgId={tvgId}
            validating={validating}
            validation={tvgIdValidation}
            isTvgIdInvalid={isTvgIdInvalid}
            originalTvgId={channel.tvg_id}
            onTvgIdChange={setTvgId}
            onTvgIdBlur={handleTvgIdBlur}
            onSuggestionClick={handleSuggestionClick}
          />

          <FormField label="TVG-Name" htmlFor="tvg-name">
            <input
              id="tvg-name"
              type="text"
              value={tvgName}
              onChange={(e) => setTvgName(e.target.value)}
              placeholder={stream.tvg_name || 'Original TVG-Name'}
            />
          </FormField>

          <FormField label="TVG-Logo URL" htmlFor="tvg-logo">
            <input
              id="tvg-logo"
              type="text"
              value={tvgLogo}
              onChange={(e) => setTvgLogo(e.target.value)}
              placeholder={channel.tvg_logo || 'Original TVG-Logo'}
            />
          </FormField>

          <FormField label="Group Title" htmlFor="group-title">
            <input
              id="group-title"
              type="text"
              value={groupTitle}
              onChange={(e) => setGroupTitle(e.target.value)}
              placeholder={channel.group_title || 'Original Group Title'}
            />
          </FormField>

          <CustomAttributesSection
            customAttributes={customAttributes}
            onAdd={handleAddCustomAttribute}
            onRemove={handleRemoveCustomAttribute}
            onChange={handleCustomAttributeChange}
          />

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
