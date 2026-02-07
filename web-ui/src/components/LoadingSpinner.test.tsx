import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { LoadingSpinner } from './LoadingSpinner'

describe('LoadingSpinner', () => {
  it('should render with default props', () => {
    const { container } = render(<LoadingSpinner />)

    const spinner = container.querySelector('.loading-spinner')
    expect(spinner).toBeInTheDocument()
    expect(spinner).toHaveClass('loading-spinner-medium')
  })

  it('should render with small size', () => {
    const { container } = render(<LoadingSpinner size="small" />)

    const spinner = container.querySelector('.loading-spinner')
    expect(spinner).toHaveClass('loading-spinner-small')
  })

  it('should render with large size', () => {
    const { container } = render(<LoadingSpinner size="large" />)

    const spinner = container.querySelector('.loading-spinner')
    expect(spinner).toHaveClass('loading-spinner-large')
  })

  it('should render inline when inline prop is true', () => {
    const { container } = render(<LoadingSpinner inline />)

    const wrapper = container.querySelector('.loading-spinner-wrapper')
    expect(wrapper).toHaveClass('inline')
  })

  it('should not be inline by default', () => {
    const { container } = render(<LoadingSpinner />)

    const wrapper = container.querySelector('.loading-spinner-wrapper')
    expect(wrapper).not.toHaveClass('inline')
  })

  it('should render four div elements for animation', () => {
    const { container } = render(<LoadingSpinner />)

    const spinner = container.querySelector('.loading-spinner')
    const divs = spinner?.querySelectorAll('div')

    expect(divs).toHaveLength(4)
  })

  it('should combine size and inline props correctly', () => {
    const { container } = render(<LoadingSpinner size="large" inline />)

    const wrapper = container.querySelector('.loading-spinner-wrapper')
    const spinner = container.querySelector('.loading-spinner')

    expect(wrapper).toHaveClass('inline')
    expect(spinner).toHaveClass('loading-spinner-large')
  })
})
