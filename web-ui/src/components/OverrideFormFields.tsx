import type { Channel } from '../types'
import type { CustomAttribute } from '../hooks/useChannelOverrideForm'
import { FormField } from './FormField'
import { TvgIdField } from './TvgIdField'
import { CustomAttributesSection } from './CustomAttributesSection'

interface OverrideFormFieldsProps {
  stream: Channel['streams'][0]
  channel: Channel
  enabled: boolean
  tvgId: string
  tvgName: string
  tvgLogo: string
  groupTitle: string
  customAttributes: CustomAttribute[]
  forceCheck: boolean
  validating: boolean
  validationError: string | null
  tvgIdValidation: { valid: boolean; suggestions: string[] } | null
  isTvgIdInvalid: boolean
  onEnabledChange: (enabled: boolean) => void
  onTvgIdChange: (value: string) => void
  onTvgNameChange: (value: string) => void
  onTvgLogoChange: (value: string) => void
  onGroupTitleChange: (value: string) => void
  onForceCheckChange: (checked: boolean) => void
  onTvgIdBlur: () => void
  onSuggestionClick: (suggestion: string) => void
  onAddCustomAttribute: () => void
  onRemoveCustomAttribute: (index: number) => void
  onCustomAttributeChange: (index: number, field: 'key' | 'value', value: string) => void
}

export function OverrideFormFields({
  stream,
  channel,
  enabled,
  tvgId,
  tvgName,
  tvgLogo,
  groupTitle,
  customAttributes,
  forceCheck,
  validating,
  validationError,
  tvgIdValidation,
  isTvgIdInvalid,
  onEnabledChange,
  onTvgIdChange,
  onTvgNameChange,
  onTvgLogoChange,
  onGroupTitleChange,
  onForceCheckChange,
  onTvgIdBlur,
  onSuggestionClick,
  onAddCustomAttribute,
  onRemoveCustomAttribute,
  onCustomAttributeChange,
}: OverrideFormFieldsProps) {
  return (
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
            onChange={(e) => onEnabledChange(e.target.checked)}
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
        onTvgIdChange={onTvgIdChange}
        onTvgIdBlur={onTvgIdBlur}
        onSuggestionClick={onSuggestionClick}
      />

      <FormField label="TVG-Name" htmlFor="tvg-name">
        <input
          id="tvg-name"
          type="text"
          value={tvgName}
          onChange={(e) => onTvgNameChange(e.target.value)}
          placeholder={stream.tvg_name || 'Original TVG-Name'}
        />
      </FormField>

      <FormField label="TVG-Logo URL" htmlFor="tvg-logo">
        <input
          id="tvg-logo"
          type="text"
          value={tvgLogo}
          onChange={(e) => onTvgLogoChange(e.target.value)}
          placeholder={channel.tvg_logo || 'Original TVG-Logo'}
        />
      </FormField>

      <FormField label="Group Title" htmlFor="group-title">
        <input
          id="group-title"
          type="text"
          value={groupTitle}
          onChange={(e) => onGroupTitleChange(e.target.value)}
          placeholder={channel.group_title || 'Original Group Title'}
        />
      </FormField>

      <CustomAttributesSection
        customAttributes={customAttributes}
        onAdd={onAddCustomAttribute}
        onRemove={onRemoveCustomAttribute}
        onChange={onCustomAttributeChange}
      />

      {isTvgIdInvalid && (
        <div className="form-field">
          <label>
            <input
              type="checkbox"
              checked={forceCheck}
              onChange={(e) => onForceCheckChange(e.target.checked)}
            />
            <span className="force-label">Force save (skip validation)</span>
          </label>
        </div>
      )}
    </div>
  )
}
