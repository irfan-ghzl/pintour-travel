import { useEffect, useState } from 'react'

export interface Countdown {
  days: number
  hours: number
  minutes: number
  seconds: number
  isExpired: boolean
}

function calc(target: number): Countdown {
  const diff = target - Date.now()
  if (diff <= 0) {
    return { days: 0, hours: 0, minutes: 0, seconds: 0, isExpired: true }
  }
  return {
    days: Math.floor(diff / 86_400_000),
    hours: Math.floor((diff % 86_400_000) / 3_600_000),
    minutes: Math.floor((diff % 3_600_000) / 60_000),
    seconds: Math.floor((diff % 60_000) / 1_000),
    isExpired: false,
  }
}

// useCountdown returns a live countdown to targetDate, ticking every second (§5.9).
export function useCountdown(targetDate: Date | string | number): Countdown {
  const target = new Date(targetDate).getTime()
  const [cd, setCd] = useState<Countdown>(() => calc(target))

  useEffect(() => {
    setCd(calc(target))
    const id = setInterval(() => setCd(calc(target)), 1000)
    return () => clearInterval(id)
  }, [target])

  return cd
}
