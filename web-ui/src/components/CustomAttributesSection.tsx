interface CustomAttribute {
  key: string
  value: string
}

interface CustomAttributesSectionProps {
  customAttributes: CustomAttribute[]
  onAdd: () => void
  onRemove: (index: number) => void
  onChange: (index: number, field: 'key' | 'value', value: string) => void
}

export function CustomAttributesSection({
  customAttributes,
  onAdd,
  onRemove,
  onChange,
}: CustomAttributesSectionProps) {
  return (
    <div className="custom-attributes-section">
      <div className="section-header">
        <h4>Custom Attributes</h4>
        <button type="button" className="add-attribute-button" onClick={onAdd}>
          + Add
        </button>
      </div>

      {customAttributes.map((attr, index) => (
        <div key={index} className="custom-attribute">
          <input
            type="text"
            placeholder="Key"
            value={attr.key}
            onChange={(e) => onChange(index, 'key', e.target.value)}
          />
          <input
            type="text"
            placeholder="Value"
            value={attr.value}
            onChange={(e) => onChange(index, 'value', e.target.value)}
          />
          <button type="button" className="remove-attribute-button" onClick={() => onRemove(index)}>
            Remove
          </button>
        </div>
      ))}

      {customAttributes.length === 0 && (
        <p className="no-attributes">No custom attributes. Click "Add" to create one.</p>
      )}
    </div>
  )
}
