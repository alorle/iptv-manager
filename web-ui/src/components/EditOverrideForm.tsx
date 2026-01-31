import { useState, useEffect } from 'react';
import type { Channel } from '../types';
import {
  updateOverride,
  deleteOverride,
  validateTvgId,
  type ValidationError,
} from '../api/channels';
import './EditOverrideForm.css';

interface EditOverrideFormProps {
  channel: Channel;
  onClose: () => void;
  onSave: () => void;
}

interface CustomAttribute {
  key: string;
  value: string;
}

export function EditOverrideForm({
  channel,
  onClose,
  onSave,
}: EditOverrideFormProps) {
  const [enabled, setEnabled] = useState<boolean>(channel.enabled);
  const [tvgId, setTvgId] = useState<string>(channel.tvg_id);
  const [tvgName, setTvgName] = useState<string>(channel.tvg_name);
  const [tvgLogo, setTvgLogo] = useState<string>(channel.tvg_logo);
  const [groupTitle, setGroupTitle] = useState<string>(channel.group_title);
  const [customAttributes, setCustomAttributes] = useState<CustomAttribute[]>(
    []
  );

  const [tvgIdValidation, setTvgIdValidation] = useState<{
    valid: boolean;
    suggestions: string[];
  } | null>(null);
  const [validating, setValidating] = useState(false);
  const [saving, setSaving] = useState(false);
  const [forceCheck, setForceCheck] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState(false);

  // Validation function
  const performValidation = async (value: string) => {
    const trimmedTvgId = value.trim();

    // Empty TVG-ID is always valid
    if (trimmedTvgId === '') {
      setTvgIdValidation({ valid: true, suggestions: [] });
      return;
    }

    // Don't validate if same as original
    if (trimmedTvgId === channel.tvg_id) {
      setTvgIdValidation({ valid: true, suggestions: [] });
      return;
    }

    setValidating(true);
    try {
      const result = await validateTvgId(trimmedTvgId);
      setTvgIdValidation({
        valid: result.valid,
        suggestions: result.suggestions || [],
      });
    } catch (err) {
      console.error('Failed to validate TVG-ID:', err);
      setTvgIdValidation({ valid: true, suggestions: [] }); // Assume valid on error
    } finally {
      setValidating(false);
    }
  };

  // Debounced TVG-ID validation while typing
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      performValidation(tvgId);
    }, 300); // 300ms debounce

    return () => clearTimeout(timeoutId);
  }, [tvgId, channel.tvg_id]);

  // Handle blur event to validate immediately
  const handleTvgIdBlur = () => {
    performValidation(tvgId);
  };

  const handleAddCustomAttribute = () => {
    setCustomAttributes([...customAttributes, { key: '', value: '' }]);
  };

  const handleRemoveCustomAttribute = (index: number) => {
    setCustomAttributes(customAttributes.filter((_, i) => i !== index));
  };

  const handleCustomAttributeChange = (
    index: number,
    field: 'key' | 'value',
    value: string
  ) => {
    const updated = [...customAttributes];
    updated[index][field] = value;
    setCustomAttributes(updated);
  };

  const handleSave = async () => {
    setError(null);
    setSaving(true);

    try {
      // Build override object with only changed fields
      const override: Record<string, unknown> = {};

      if (enabled !== channel.enabled) {
        override.enabled = enabled;
      }

      if (tvgId.trim() !== channel.tvg_id) {
        override.tvg_id = tvgId.trim() || null;
      }

      if (tvgName.trim() !== channel.tvg_name) {
        override.tvg_name = tvgName.trim() || null;
      }

      if (tvgLogo.trim() !== channel.tvg_logo) {
        override.tvg_logo = tvgLogo.trim() || null;
      }

      if (groupTitle.trim() !== channel.group_title) {
        override.group_title = groupTitle.trim() || null;
      }

      // Add custom attributes (currently not supported by backend, placeholder)
      if (customAttributes.length > 0) {
        const customAttrs: Record<string, string> = {};
        customAttributes.forEach((attr) => {
          if (attr.key.trim()) {
            customAttrs[attr.key.trim()] = attr.value;
          }
        });
        if (Object.keys(customAttrs).length > 0) {
          // Store as a comment for now since backend doesn't support it yet
          console.log('Custom attributes not yet supported:', customAttrs);
        }
      }

      await updateOverride(channel.acestream_id, override, forceCheck);
      onSave();
    } catch (err) {
      if (err && typeof err === 'object' && 'error' in err) {
        const validationErr = err as ValidationError;
        setError(validationErr.message || 'Validation failed');
        if (validationErr.suggestions) {
          setTvgIdValidation({
            valid: false,
            suggestions: validationErr.suggestions,
          });
        }
      } else {
        setError(
          err instanceof Error ? err.message : 'Failed to save override'
        );
      }
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteConfirm) {
      setDeleteConfirm(true);
      return;
    }

    setError(null);
    setSaving(true);

    try {
      await deleteOverride(channel.acestream_id);
      onSave();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete override');
    } finally {
      setSaving(false);
      setDeleteConfirm(false);
    }
  };

  const handleSuggestionClick = (suggestion: string) => {
    setTvgId(suggestion);
  };

  const isTvgIdInvalid =
    tvgIdValidation && !tvgIdValidation.valid && tvgId.trim() !== '';
  const canSave = !validating && (!isTvgIdInvalid || forceCheck);

  return (
    <div className="edit-override-form-overlay" onClick={onClose}>
      <div
        className="edit-override-form"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="form-header">
          <h2>Edit Channel Override</h2>
          <button className="close-button" onClick={onClose}>
            ×
          </button>
        </div>

        <div className="form-content">
          <div className="channel-info">
            <h3>{channel.name}</h3>
            <p className="acestream-id">ID: {channel.acestream_id}</p>
          </div>

          {error && <div className="error-message">{error}</div>}

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
            {validating && (
              <span className="validation-status validating">
                Validating...
              </span>
            )}
            {!validating && tvgIdValidation && tvgId.trim() !== '' && (
              <span
                className={`validation-status ${
                  tvgIdValidation.valid ? 'valid' : 'invalid'
                }`}
              >
                {tvgIdValidation.valid ? '✓ Valid' : '✗ Invalid'}
              </span>
            )}
            {isTvgIdInvalid && tvgIdValidation.suggestions.length > 0 && (
              <div className="suggestions">
                <p className="suggestions-label">Did you mean:</p>
                <ul>
                  {tvgIdValidation.suggestions.map((suggestion) => (
                    <li
                      key={suggestion}
                      onClick={() => handleSuggestionClick(suggestion)}
                    >
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
              placeholder={channel.tvg_name || 'Original TVG-Name'}
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
                  onChange={(e) =>
                    handleCustomAttributeChange(index, 'key', e.target.value)
                  }
                />
                <input
                  type="text"
                  placeholder="Value"
                  value={attr.value}
                  onChange={(e) =>
                    handleCustomAttributeChange(index, 'value', e.target.value)
                  }
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
              <p className="no-attributes">
                No custom attributes. Click "Add" to create one.
              </p>
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
                <span className="force-label">
                  Force save (skip validation)
                </span>
              </label>
            </div>
          )}
        </div>

        <div className="form-actions">
          <button
            className="save-button"
            onClick={handleSave}
            disabled={saving || !canSave}
          >
            {saving ? 'Saving...' : 'Save'}
          </button>
          <button className="cancel-button" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          {channel.has_override && (
            <button
              className={`delete-button ${deleteConfirm ? 'confirm' : ''}`}
              onClick={handleDelete}
              disabled={saving}
            >
              {deleteConfirm ? 'Confirm Delete?' : 'Delete Override'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
