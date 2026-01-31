import { useEffect, useState } from 'react'
import type { Channel } from '../types'
import { listChannels } from '../api/channels'

export function useChannels(refreshTrigger?: number) {
  const [channels, setChannels] = useState<Channel[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const fetchChannels = async () => {
    try {
      setLoading(true)
      setError(null)
      const data = await listChannels()
      setChannels(data)
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchChannels()
  }, [refreshTrigger])

  return { channels, loading, error, refetch: fetchChannels }
}
