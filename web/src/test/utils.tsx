/* eslint-disable react-refresh/only-export-components */
import { ReactElement, ReactNode } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom'

interface WrapperProps {
  children: ReactNode
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  routerProps?: MemoryRouterProps
}

function createWrapper(routerProps?: MemoryRouterProps) {
  return function Wrapper({ children }: WrapperProps) {
    return <MemoryRouter {...routerProps}>{children}</MemoryRouter>
  }
}

export function renderWithRouter(
  ui: ReactElement,
  { routerProps, ...renderOptions }: CustomRenderOptions = {}
) {
  return render(ui, {
    wrapper: createWrapper(routerProps),
    ...renderOptions,
  })
}

export * from '@testing-library/react'
export { renderWithRouter as render }
