interface BulkActionsBarProps {
  selectedCount: number
  onBulkEdit: () => void
}

export function BulkActionsBar({ selectedCount, onBulkEdit }: BulkActionsBarProps) {
  if (selectedCount === 0) return null

  return (
    <div className="bulk-actions" role="toolbar" aria-label="Bulk actions">
      <div className="selection-info" aria-live="polite" aria-atomic="true">
        {selectedCount} channel(s) selected
      </div>
      <button type="button" className="button button-primary" onClick={onBulkEdit}>
        Bulk Edit
      </button>
    </div>
  )
}
