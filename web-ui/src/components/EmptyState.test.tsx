import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { EmptyState } from './EmptyState'

describe('EmptyState', () => {
  const defaultProps = {
    enabledFilter: 'all' as const,
    totalChannelsCount: 10,
    searchText: '',
    groupFilter: '',
  }

  it('should render as table row', () => {
    const { container } = render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} />
        </tbody>
      </table>
    )

    const tr = container.querySelector('tr')
    expect(tr).toBeInTheDocument()
  })

  it('should render td with colspan 5', () => {
    const { container } = render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} />
        </tbody>
      </table>
    )

    const td = container.querySelector('td')
    expect(td).toHaveAttribute('colSpan', '5')
  })

  it('should show message for enabled filter with channels', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} enabledFilter="enabled" totalChannelsCount={5} />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels with enabled streams found')).toBeInTheDocument()
  })

  it('should show message for disabled filter with channels', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} enabledFilter="disabled" totalChannelsCount={5} />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels with all streams disabled found')).toBeInTheDocument()
  })

  it('should show message when search text is active', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} searchText="BBC" />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels match the current filters')).toBeInTheDocument()
  })

  it('should show message when group filter is active', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} groupFilter="UK" />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels match the current filters')).toBeInTheDocument()
  })

  it('should show message when both search and group filters are active', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} searchText="BBC" groupFilter="UK" />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels match the current filters')).toBeInTheDocument()
  })

  it('should show generic message when no channels exist', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} totalChannelsCount={0} />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels found')).toBeInTheDocument()
  })

  it('should prioritize enabled filter message over search filter', () => {
    render(
      <table>
        <tbody>
          <EmptyState
            {...defaultProps}
            enabledFilter="enabled"
            totalChannelsCount={5}
            searchText="BBC"
          />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels with enabled streams found')).toBeInTheDocument()
    expect(screen.queryByText('No channels match the current filters')).not.toBeInTheDocument()
  })

  it('should prioritize disabled filter message over search filter', () => {
    render(
      <table>
        <tbody>
          <EmptyState
            {...defaultProps}
            enabledFilter="disabled"
            totalChannelsCount={5}
            searchText="BBC"
          />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels with all streams disabled found')).toBeInTheDocument()
    expect(screen.queryByText('No channels match the current filters')).not.toBeInTheDocument()
  })

  it('should have empty-state CSS class', () => {
    const { container } = render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} />
        </tbody>
      </table>
    )

    const td = container.querySelector('td')
    expect(td).toHaveClass('empty-state')
  })

  it('should show generic message for all filter with no channels', () => {
    render(
      <table>
        <tbody>
          <EmptyState {...defaultProps} enabledFilter="all" totalChannelsCount={0} />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels found')).toBeInTheDocument()
  })

  it('should show filter message when enabled filter is all but search is active', () => {
    render(
      <table>
        <tbody>
          <EmptyState
            {...defaultProps}
            enabledFilter="all"
            totalChannelsCount={10}
            searchText="test"
          />
        </tbody>
      </table>
    )

    expect(screen.getByText('No channels match the current filters')).toBeInTheDocument()
  })
})
