interface EmptyStateProps {
  enabledFilter: 'all' | 'enabled' | 'disabled'
  totalChannelsCount: number
  searchText: string
  groupFilter: string
}

export function EmptyState({
  enabledFilter,
  totalChannelsCount,
  searchText,
  groupFilter,
}: EmptyStateProps) {
  let message: string

  if (enabledFilter === 'enabled' && totalChannelsCount > 0) {
    message = 'No channels with enabled streams found'
  } else if (enabledFilter === 'disabled' && totalChannelsCount > 0) {
    message = 'No channels with all streams disabled found'
  } else if (searchText || groupFilter) {
    message = 'No channels match the current filters'
  } else {
    message = 'No channels found'
  }

  return (
    <tr>
      <td colSpan={5} className="empty-state">
        {message}
      </td>
    </tr>
  )
}
