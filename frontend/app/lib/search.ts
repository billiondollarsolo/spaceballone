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

const EMPTY_RESULTS: SearchResult[] = []

export function useSearch(query: string, debounceMs = 300) {
  const [searchState, setSearchState] = useState<{
    results: SearchResult[]
    fetchedQuery: string
  }>({ results: EMPTY_RESULTS, fetchedQuery: '' })
  const abortRef = useRef<AbortController | null>(null)
  const trimmedQuery = query?.trim() ?? ''
  const isValidQuery = trimmedQuery.length >= 2

  useEffect(() => {
    if (!isValidQuery) {
      return
    }

    const timeout = setTimeout(async () => {
      // Cancel previous request
      abortRef.current?.abort()
      const controller = new AbortController()
      abortRef.current = controller

      try {
        const data = await searchApi.search(trimmedQuery, controller.signal)
        if (!controller.signal.aborted) {
          setSearchState({ results: data, fetchedQuery: trimmedQuery })
        }
      } catch {
        if (!controller.signal.aborted) {
          setSearchState({ results: EMPTY_RESULTS, fetchedQuery: trimmedQuery })
        }
      }
    }, debounceMs)

    return () => {
      clearTimeout(timeout)
    }
  }, [trimmedQuery, isValidQuery, debounceMs])

  // Derive loading and results from state + current query
  const results = isValidQuery ? searchState.results : EMPTY_RESULTS
  const isLoading = isValidQuery && searchState.fetchedQuery !== trimmedQuery

  const grouped = useMemo(() => ({
    machines: results.filter((r) => r.type === 'machine'),
    projects: results.filter((r) => r.type === 'project'),
    sessions: results.filter((r) => r.type === 'session'),
  }), [results])

  return { results, grouped, isLoading }
}
