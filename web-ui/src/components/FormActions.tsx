import { LoadingSpinner } from './LoadingSpinner'

interface FormActionsProps {
  saving: boolean
  canSave: boolean
  hasOverride: boolean
  onSave: () => void
  onCancel: () => void
  onDelete: () => void
}

export function FormActions({
  saving,
  canSave,
  hasOverride,
  onSave,
  onCancel,
  onDelete,
}: FormActionsProps) {
  return (
    <div className="form-actions">
      <button className="save-button" onClick={onSave} disabled={saving || !canSave}>
        {saving ? (
          <>
            Saving
            <LoadingSpinner size="small" inline />
          </>
        ) : (
          'Save'
        )}
      </button>
      <button className="cancel-button" onClick={onCancel} disabled={saving}>
        Cancel
      </button>
      {hasOverride && (
        <button className="delete-button" onClick={onDelete} disabled={saving}>
          Delete Override
        </button>
      )}
    </div>
  )
}
