import { useCallback, useEffect, useState } from 'react'
import { fetchProfile, type ProfileData } from '@/api/profile'

export function useProfile() {
  const [data, setData] = useState<ProfileData | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<unknown>(null)

  const reload = useCallback(async () => {
    setError(null)
    try {
      setData(await fetchProfile())
    } catch (e) {
      setError(e)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  return { data, loading, error, reload }
}
