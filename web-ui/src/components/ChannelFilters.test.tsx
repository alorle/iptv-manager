import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChannelFilters } from './ChannelFilters'

describe('ChannelFilters', () => {
  const mockProps = {
    searchText: '',
    groupFilter: '',
    enabledFilter: 'enabled' as const,
    uniqueGroups: ['UK', 'Spain', 'France'],
    onSearchChange: vi.fn(),
    onGroupFilterChange: vi.fn(),
    onEnabledFilterChange: vi.fn(),
  }

  it('should render search input', () => {
    render(<ChannelFilters {...mockProps} />)

    const searchInput = screen.getByRole('searchbox', { name: /search channels/i })
    expect(searchInput).toBeInTheDocument()
  })

  it('should render group filter dropdown', () => {
    render(<ChannelFilters {...mockProps} />)

    const groupSelect = screen.getByRole('combobox', { name: /filter by group/i })
    expect(groupSelect).toBeInTheDocument()
  })

  it('should render enabled filter buttons', () => {
    render(<ChannelFilters {...mockProps} />)

    expect(screen.getByRole('button', { name: /^all$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^enabled$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^disabled$/i })).toBeInTheDocument()
  })

  it('should call onSearchChange when typing in search input', async () => {
    const user = userEvent.setup()
    render(<ChannelFilters {...mockProps} />)

    const searchInput = screen.getByRole('searchbox', { name: /search channels/i })
    await user.type(searchInput, 'A')

    // Verify onSearchChange was called
    expect(mockProps.onSearchChange).toHaveBeenCalled()
  })

  it('should call onGroupFilterChange when selecting a group', async () => {
    const user = userEvent.setup()
    render(<ChannelFilters {...mockProps} />)

    const groupSelect = screen.getByRole('combobox', { name: /filter by group/i })
    await user.selectOptions(groupSelect, 'UK')

    expect(mockProps.onGroupFilterChange).toHaveBeenCalledWith('UK')
  })

  it('should call onEnabledFilterChange when clicking filter buttons', async () => {
    const user = userEvent.setup()
    render(<ChannelFilters {...mockProps} />)

    const allButton = screen.getByRole('button', { name: /^all$/i })
    await user.click(allButton)

    expect(mockProps.onEnabledFilterChange).toHaveBeenCalledWith('all')
  })

  it('should display all groups in dropdown', () => {
    render(<ChannelFilters {...mockProps} />)

    const groupSelect = screen.getByRole('combobox', { name: /filter by group/i })
    const options = Array.from(groupSelect.querySelectorAll('option'))

    expect(options).toHaveLength(4) // "All Groups" + 3 groups
    expect(options[0]).toHaveTextContent('All Groups')
    expect(options[1]).toHaveTextContent('UK')
    expect(options[2]).toHaveTextContent('Spain')
    expect(options[3]).toHaveTextContent('France')
  })

  it('should show active state on enabled filter button', () => {
    render(<ChannelFilters {...mockProps} />)

    const enabledButton = screen.getByRole('button', { name: /^enabled$/i })
    expect(enabledButton).toHaveClass('active')
    expect(enabledButton).toHaveAttribute('aria-pressed', 'true')
  })

  it('should show active state on correct button based on enabledFilter prop', () => {
    render(<ChannelFilters {...mockProps} enabledFilter="all" />)

    const allButton = screen.getByRole('button', { name: /^all$/i })
    const enabledButton = screen.getByRole('button', { name: /^enabled$/i })

    expect(allButton).toHaveClass('active')
    expect(allButton).toHaveAttribute('aria-pressed', 'true')
    expect(enabledButton).not.toHaveClass('active')
    expect(enabledButton).toHaveAttribute('aria-pressed', 'false')
  })

  it('should display current search text value', () => {
    render(<ChannelFilters {...mockProps} searchText="test query" />)

    const searchInput = screen.getByRole('searchbox', { name: /search channels/i })
    expect(searchInput).toHaveValue('test query')
  })

  it('should display current group filter value', () => {
    render(<ChannelFilters {...mockProps} groupFilter="UK" />)

    const groupSelect = screen.getByRole('combobox', { name: /filter by group/i })
    expect(groupSelect).toHaveValue('UK')
  })

  it('should have proper accessibility attributes', () => {
    render(<ChannelFilters {...mockProps} />)

    const searchInput = screen.getByRole('searchbox', { name: /search channels/i })
    expect(searchInput).toHaveAttribute('aria-describedby', 'search-results')

    const filterGroup = screen.getByRole('group', { name: /filter by enabled status/i })
    expect(filterGroup).toBeInTheDocument()
  })

  it('should handle empty groups array', () => {
    render(<ChannelFilters {...mockProps} uniqueGroups={[]} />)

    const groupSelect = screen.getByRole('combobox', { name: /filter by group/i })
    const options = Array.from(groupSelect.querySelectorAll('option'))

    expect(options).toHaveLength(1) // Only "All Groups"
    expect(options[0]).toHaveTextContent('All Groups')
  })

  it('should call all three filter change handlers independently', async () => {
    const user = userEvent.setup()
    const onSearchChange = vi.fn()
    const onGroupFilterChange = vi.fn()
    const onEnabledFilterChange = vi.fn()

    render(
      <ChannelFilters
        {...mockProps}
        onSearchChange={onSearchChange}
        onGroupFilterChange={onGroupFilterChange}
        onEnabledFilterChange={onEnabledFilterChange}
      />
    )

    const searchInput = screen.getByRole('searchbox', { name: /search channels/i })
    await user.type(searchInput, 'A')
    expect(onSearchChange).toHaveBeenCalled()

    const groupSelect = screen.getByRole('combobox', { name: /filter by group/i })
    await user.selectOptions(groupSelect, 'UK')
    expect(onGroupFilterChange).toHaveBeenCalled()

    const disabledButton = screen.getByRole('button', { name: /^disabled$/i })
    await user.click(disabledButton)
    expect(onEnabledFilterChange).toHaveBeenCalled()
  })

  it('should have visually hidden labels for accessibility', () => {
    const { container } = render(<ChannelFilters {...mockProps} />)

    const labels = container.querySelectorAll('.visually-hidden')
    expect(labels.length).toBeGreaterThan(0)
  })
})
