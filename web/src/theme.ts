import { useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'esteira-tema'

function systemTheme(): Theme {
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function savedTheme(): Theme | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    return value === 'light' || value === 'dark' ? value : null
  } catch {
    return null
  }
}

// useTheme começa pelo tema salvo (ou o do sistema) e aplica em <html data-theme>.
export function useTheme() {
  const [theme, setTheme] = useState<Theme>(() => savedTheme() ?? systemTheme())

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      // Sem armazenamento (modo privado): o tema vale só para esta visita.
    }
  }, [theme])

  const toggle = () => setTheme((t) => (t === 'dark' ? 'light' : 'dark'))
  return { theme, toggle }
}
