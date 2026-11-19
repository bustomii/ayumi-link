import { useEffect, useRef } from 'react'

/** Ref yang selalu berisi nilai terbaru; dibaca dari callback/async, bukan saat render. */
export function useLatest<T>(value: T) {
  const ref = useRef(value)
  useEffect(() => {
    ref.current = value
  })
  return ref
}
