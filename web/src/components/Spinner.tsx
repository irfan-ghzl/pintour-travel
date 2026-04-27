import { Loader2 } from 'lucide-react'

export default function Spinner({ message }: { message?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-20 gap-3 text-gray-400">
      <Loader2 className="w-8 h-8 animate-spin text-primary-500" />
      {message && <p className="text-sm">{message}</p>}
    </div>
  )
}
