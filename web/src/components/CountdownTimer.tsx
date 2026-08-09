import clsx from 'clsx'
import { useCountdown } from '../hooks/useCountdown'

// CountdownTimer renders a live D/H/M/S countdown to a target date (§5.8).
export default function CountdownTimer({
  target,
  className,
  expiredText = 'Sudah berlalu',
  compact = false,
}: {
  target: Date | string | number
  className?: string
  expiredText?: string
  compact?: boolean
}) {
  const { days, hours, minutes, seconds, isExpired } = useCountdown(target)

  if (isExpired) {
    return <span className={clsx('text-gray-400', className)}>{expiredText}</span>
  }

  if (compact) {
    return (
      <span className={clsx('font-medium', className)}>
        {days > 0 ? `${days} hari lagi` : `${hours}j ${minutes}m`}
      </span>
    )
  }

  const cells = [
    { v: days, l: 'Hari' },
    { v: hours, l: 'Jam' },
    { v: minutes, l: 'Menit' },
    { v: seconds, l: 'Detik' },
  ]
  return (
    <div className={clsx('flex gap-2', className)}>
      {cells.map(({ v, l }) => (
        <div key={l} className="flex flex-col items-center bg-gray-50 rounded-lg px-3 py-2 min-w-[3rem]">
          <span className="text-xl font-bold text-gray-800 tabular-nums">{String(v).padStart(2, '0')}</span>
          <span className="text-[10px] uppercase text-gray-400">{l}</span>
        </div>
      ))}
    </div>
  )
}
