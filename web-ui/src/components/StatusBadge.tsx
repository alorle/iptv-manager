interface StatusBadgeProps {
  type: 'disabled' | 'override'
  label?: string
}

export function StatusBadge({ type, label }: StatusBadgeProps) {
  if (type === 'disabled') {
    return (
      <span className="disabled-badge" aria-label={label || 'Disabled'}>
        Disabled
      </span>
    )
  }

  return (
    <span
      className="override-indicator-inline"
      title={label || 'Has custom overrides'}
      aria-label={label || 'Has custom overrides'}
    >
      ⚙
    </span>
  )
}
