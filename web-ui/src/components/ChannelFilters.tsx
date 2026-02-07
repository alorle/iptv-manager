interface ChannelFiltersProps {
  searchText: string
  groupFilter: string
  enabledFilter: 'all' | 'enabled' | 'disabled'
  uniqueGroups: string[]
  onSearchChange: (value: string) => void
  onGroupFilterChange: (value: string) => void
  onEnabledFilterChange: (value: 'all' | 'enabled' | 'disabled') => void
}

export function ChannelFilters({
  searchText,
  groupFilter,
  enabledFilter,
  uniqueGroups,
  onSearchChange,
  onGroupFilterChange,
  onEnabledFilterChange,
}: ChannelFiltersProps) {
  return (
    <div className="filters" role="search" aria-label="Filter channels">
      <label htmlFor="channel-search" className="visually-hidden">
        Search channels
      </label>
      <input
        id="channel-search"
        type="search"
        className="search-input"
        placeholder="Search channels..."
        value={searchText}
        onChange={(e) => onSearchChange(e.target.value)}
        aria-describedby="search-results"
      />
      <label htmlFor="group-filter" className="visually-hidden">
        Filter by group
      </label>
      <select
        id="group-filter"
        className="group-filter"
        value={groupFilter}
        onChange={(e) => onGroupFilterChange(e.target.value)}
      >
        <option value="">All Groups</option>
        {uniqueGroups.map((group) => (
          <option key={group} value={group}>
            {group}
          </option>
        ))}
      </select>
      <div className="enabled-filter" role="group" aria-label="Filter by enabled status">
        <button
          type="button"
          className={`filter-button ${enabledFilter === 'all' ? 'active' : ''}`}
          onClick={() => onEnabledFilterChange('all')}
          aria-pressed={enabledFilter === 'all'}
        >
          All
        </button>
        <button
          type="button"
          className={`filter-button ${enabledFilter === 'enabled' ? 'active' : ''}`}
          onClick={() => onEnabledFilterChange('enabled')}
          aria-pressed={enabledFilter === 'enabled'}
        >
          Enabled
        </button>
        <button
          type="button"
          className={`filter-button ${enabledFilter === 'disabled' ? 'active' : ''}`}
          onClick={() => onEnabledFilterChange('disabled')}
          aria-pressed={enabledFilter === 'disabled'}
        >
          Disabled
        </button>
      </div>
    </div>
  )
}
