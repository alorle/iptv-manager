interface TvgIdFieldProps {
  tvgId: string
  validating: boolean
  validation: { valid: boolean; suggestions: string[] } | null
  isTvgIdInvalid: boolean
  originalTvgId: string
  onTvgIdChange: (value: string) => void
  onTvgIdBlur: () => void
  onSuggestionClick: (suggestion: string) => void
}

export function TvgIdField({
  tvgId,
  validating,
  validation,
  isTvgIdInvalid,
  originalTvgId,
  onTvgIdChange,
  onTvgIdBlur,
  onSuggestionClick,
}: TvgIdFieldProps) {
  return (
    <div className="form-field">
      <label htmlFor="tvg-id">TVG-ID</label>
      <input
        id="tvg-id"
        type="text"
        value={tvgId}
        onChange={(e) => onTvgIdChange(e.target.value)}
        onBlur={onTvgIdBlur}
        placeholder={originalTvgId || 'Original TVG-ID'}
      />
      {validating && <span className="validation-status validating">Validating...</span>}
      {!validating && validation && tvgId.trim() !== '' && (
        <span className={`validation-status ${validation.valid ? 'valid' : 'invalid'}`}>
          {validation.valid ? '✓ Valid' : '✗ Invalid'}
        </span>
      )}
      {isTvgIdInvalid && validation && validation.suggestions.length > 0 && (
        <div className="suggestions">
          <p className="suggestions-label">Did you mean:</p>
          <ul>
            {validation.suggestions.map((suggestion) => (
              <li key={suggestion} onClick={() => onSuggestionClick(suggestion)}>
                {suggestion}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
