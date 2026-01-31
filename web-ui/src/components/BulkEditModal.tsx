import { useState } from 'react';
import './BulkEditModal.css';

interface BulkEditModalProps {
  selectedCount: number;
  onClose: () => void;
  onSubmit: (field: string, value: string | boolean) => Promise<void>;
}

type BulkEditField = 'enabled' | 'tvg_id' | 'tvg_name' | 'tvg_logo' | 'group_title';

export function BulkEditModal({ selectedCount, onClose, onSubmit }: BulkEditModalProps) {
  const [field, setField] = useState<BulkEditField>('enabled');
  const [value, setValue] = useState<string>('true');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);

    try {
      let processedValue: string | boolean = value;

      // Convert enabled field to boolean
      if (field === 'enabled') {
        processedValue = value === 'true';
      }

      await onSubmit(field, processedValue);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update channels');
    } finally {
      setSubmitting(false);
    }
  };

  // Reset value when field changes
  const handleFieldChange = (newField: BulkEditField) => {
    setField(newField);
    // Set default value based on field type
    setValue(newField === 'enabled' ? 'true' : '');
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Bulk Edit Channels</h2>
          <button className="close-button" onClick={onClose}>
            ×
          </button>
        </div>

        <form onSubmit={handleSubmit}>
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
                This will update <strong>{selectedCount}</strong> channel{selectedCount !== 1 ? 's' : ''}
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
            <button
              type="submit"
              className="button button-primary"
              disabled={submitting}
            >
              {submitting ? 'Updating...' : 'Update Channels'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
