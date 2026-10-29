import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// O jsdom não calcula layout, então as animações de layout do Motion nunca
// terminam. Nos testes simulamos "reduzir movimento" (que a interface respeita),
// e o Motion aplica as mudanças sem animar. Nenhuma outra media query casa.
window.matchMedia = (query: string) =>
  ({
    matches: query.includes('prefers-reduced-motion: reduce'),
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  }) as MediaQueryList

afterEach(() => {
  cleanup()
})
