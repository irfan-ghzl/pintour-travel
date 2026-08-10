import clsx from 'clsx'

// ProgressBar renders a horizontal progress bar with an optional label (§5.8).
export default function ProgressBar({
  value,
  max = 100,
  label,
  color = 'bg-emerald-500',
  className,
}: {
  value: number
  max?: number
  label?: string
  color?: string
  className?: string
}) {
  const pct = max > 0 ? Math.min(100, Math.round((value / max) * 100)) : 0
  return (
    <div className={className}>
      {label && (
        <div className="flex justify-between text-xs text-gray-600 mb-1">
          <span>{label}</span>
          <span>{pct}%</span>
        </div>
      )}
      <div className="w-full h-2.5 bg-gray-200 rounded-full overflow-hidden">
        <div className={clsx('h-full rounded-full transition-all', color)} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}
