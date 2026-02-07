import { useState, useCallback, useRef } from 'react'
import type { Channel } from '../types'
import { useChannelOverrideForm } from '../hooks/useChannelOverrideForm'
import { useTvgIdValidation } from '../hooks/useTvgIdValidation'
import { useChannelOverrideMutations } from '../hooks/useChannelOverrideMutations'
import { useFocusManagement } from '../hooks/useFocusManagement'
import { ConfirmDialog } from './ConfirmDialog'
import { FormHeader } from './FormHeader'
import { FormActions } from './FormActions'
import { OverrideFormFields } from './OverrideFormFields'
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
        <FormHeader
          title="Edit Channel Override"
          onClose={onClose}
          closeButtonRef={closeButtonRef}
        />

        <OverrideFormFields
          stream={stream}
          channel={channel}
          enabled={enabled}
          tvgId={tvgId}
          tvgName={tvgName}
          tvgLogo={tvgLogo}
          groupTitle={groupTitle}
          customAttributes={customAttributes}
          forceCheck={forceCheck}
          validating={validating}
          validationError={validationError}
          tvgIdValidation={tvgIdValidation}
          isTvgIdInvalid={isTvgIdInvalid}
          onEnabledChange={setEnabled}
          onTvgIdChange={setTvgId}
          onTvgNameChange={setTvgName}
          onTvgLogoChange={setTvgLogo}
          onGroupTitleChange={setGroupTitle}
          onForceCheckChange={setForceCheck}
          onTvgIdBlur={handleTvgIdBlur}
          onSuggestionClick={handleSuggestionClick}
          onAddCustomAttribute={handleAddCustomAttribute}
          onRemoveCustomAttribute={handleRemoveCustomAttribute}
          onCustomAttributeChange={handleCustomAttributeChange}
        />

        <FormActions
          saving={saving}
          canSave={canSave}
          hasOverride={stream.has_override}
          onSave={handleSave}
          onCancel={onClose}
          onDelete={() => setShowDeleteConfirm(true)}
        />
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
