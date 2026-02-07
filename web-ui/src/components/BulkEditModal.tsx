import { useBulkEditForm, type BulkEditField } from '../hooks/useBulkEditForm'
import { LoadingSpinner } from './LoadingSpinner'
import './BulkEditModal.css'

interface BulkEditModalProps {
  selectedCount: number
  onClose: () => void
  onSubmit: (field: string, value: string | boolean) => Promise<void>
}

export function BulkEditModal({ selectedCount, onClose, onSubmit }: BulkEditModalProps) {
  const { field, value, submitting, error, handleFieldChange, setValue, handleSubmit } =
    useBulkEditForm()

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Bulk Edit Channels</h2>
          <button className="close-button" onClick={onClose}>
            ×
          </button>
        </div>

        <form onSubmit={(e) => handleSubmit(e, onSubmit, onClose)}>
          <div className="modal-body">
            <div className="form-group">
              <label htmlFor="field-select">Field to Update</label>
              <select
                id="field-select"
                value={field}
                onChange={(e) => handleFieldChange(e.target.value as BulkEditField)}
                disabled={submitting}
              >
                <option value="enabled">Enabled</option>
                <option value="tvg_id">TVG-ID</option>
                <option value="tvg_name">TVG Name</option>
                <option value="tvg_logo">TVG Logo</option>
                <option value="group_title">Group Title</option>
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="value-input">
                {field === 'enabled' ? 'New Status' : 'New Value'}
              </label>
              {field === 'enabled' ? (
                <select
                  id="value-input"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  disabled={submitting}
                >
                  <option value="true">Enabled</option>
                  <option value="false">Disabled</option>
                </select>
              ) : (
                <input
                  id="value-input"
                  type="text"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder={`Enter ${field.replace('_', ' ')}`}
                  disabled={submitting}
                  required={field !== 'tvg_id'}
                />
              )}
            </div>

            <div className="preview-section">
              <p className="preview-text">
                This will update <strong>{selectedCount}</strong> channel
                {selectedCount !== 1 ? 's' : ''}
              </p>
            </div>

            {error && <div className="error-message">{error}</div>}
          </div>

          <div className="modal-footer">
            <button
              type="button"
              className="button button-secondary"
              onClick={onClose}
              disabled={submitting}
            >
              Cancel
            </button>
            <button type="submit" className="button button-primary" disabled={submitting}>
              {submitting ? (
                <>
                  Updating
                  <LoadingSpinner size="small" inline />
                </>
              ) : (
                'Update Channels'
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
