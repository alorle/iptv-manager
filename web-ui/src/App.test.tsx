import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import App from './App'

describe('App', () => {
  it('renders without crashing', () => {
    const { container } = render(<App />)
    expect(container).toBeTruthy()
  })

  it('renders app container', () => {
    const { container } = render(<App />)
    const appDiv = container.querySelector('.app')
    expect(appDiv).toBeInTheDocument()
  })
})
