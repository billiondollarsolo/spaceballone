import { useState, useEffect, useRef, useMemo } from 'react'
import { api } from './api'

export interface SearchResult {
  type: 'machine' | 'project' | 'session'
  id: string
  name: string
  metadata?: Record<string, unknown>
}

export const searchApi = {
  search(q: string, signal?: AbortSignal) {
    return api.get<SearchResult[]>(`/api/search?q=${encodeURIComponent(q)}`, { signal })
  },
}

export function useSearch(query: string, debounceMs = 300) {
  const [results, setResults] = useState<SearchResult[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (!query || query.trim().length < 2) {
      setResults([])
      setIsLoading(false)
      return
    }

    setIsLoading(true)

    const timeout = setTimeout(async () => {
      // Cancel previous request
      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller

      try {
        const data = await searchApi.search(query.trim(), controller.signal)
        if (!controller.signal.aborted) {
          setResults(data)
          setIsLoading(false)
        }
      } catch {
        if (!controller.signal.aborted) {
          setResults([])
          setIsLoading(false)
        }
      }
    }, debounceMs)

    return () => {
      clearTimeout(timeout)
    }
  }, [query, debounceMs])

  const grouped = useMemo(() => ({
    machines: results.filter((r) => r.type === 'machine'),
    projects: results.filter((r) => r.type === 'project'),
    sessions: results.filter((r) => r.type === 'session'),
  }), [results])

  return { results, grouped, isLoading }
}
