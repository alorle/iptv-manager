interface ChannelTableHeaderProps {
  allSelected: boolean
  someSelected: boolean
  onSelectAll: (checked: boolean) => void
}

export function ChannelTableHeader({
  allSelected,
  someSelected,
  onSelectAll,
}: ChannelTableHeaderProps) {
  return (
    <thead>
      <tr>
        <th className="checkbox-column" scope="col">
          <input
            type="checkbox"
            checked={allSelected}
            ref={(el) => {
              if (el) el.indeterminate = someSelected
            }}
            onChange={(e) => onSelectAll(e.target.checked)}
            aria-label={allSelected ? 'Deselect all channels' : 'Select all channels'}
          />
        </th>
        <th className="logo-column" scope="col">
          Logo
        </th>
        <th scope="col">Name</th>
        <th scope="col">Group</th>
        <th scope="col">Streams</th>
      </tr>
    </thead>
  )
}
