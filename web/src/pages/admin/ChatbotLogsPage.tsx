// TODO(chatbot-toggle): tambah toggle on/off chatbot di halaman ini (switch button di header)
// yang hit PATCH /admin/chatbot/toggle — perlu endpoint baru di backend juga.
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Bot, Search, MessageSquare, UserPlus, X } from 'lucide-react'
import { Link } from 'react-router-dom'
import toast from 'react-hot-toast'
import api, {
  getChatbotConversations, getChatbotConversation, createLeadFromChat,
  type ChatbotConversation,
} from '../../utils/api'

export default function ChatbotLogsPage() {
  const [phone, setPhone] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<ChatbotConversation | null>(null)
  const [leadFor, setLeadFor] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['chatbot-logs', phone, from, to, page],
    queryFn: () => getChatbotConversations({ phone, from, to, page, limit: 20 }),
  })

  const conversations: ChatbotConversation[] = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <Bot className="text-emerald-600" size={20} />
        <h2 className="text-lg font-semibold text-gray-800">Chatbot Logs</h2>
      </div>

      {/* Filter bar */}
      <div className="flex gap-3 flex-wrap items-center">
        <div className="relative">
          <Search className="absolute left-3 top-2.5 text-gray-400" size={14} />
          <input
            className="pl-8 pr-4 py-2 border rounded-lg text-sm w-52"
            placeholder="Cari nomor HP..."
            value={phone}
            onChange={(e) => { setPhone(e.target.value); setPage(1) }}
          />
        </div>
        <div className="flex items-center gap-1">
          <input type="date" className="border rounded-lg px-2 py-2 text-sm" value={from}
            onChange={(e) => { setFrom(e.target.value); setPage(1) }} title="Dari" />
          <span className="text-xs text-gray-400">–</span>
          <input type="date" className="border rounded-lg px-2 py-2 text-sm" value={to}
            onChange={(e) => { setTo(e.target.value); setPage(1) }} title="Sampai" />
        </div>
        {(phone || from || to) && (
          <button onClick={() => { setPhone(''); setFrom(''); setTo(''); setPage(1) }}
            className="text-xs text-emerald-600 hover:underline">× Reset</button>
        )}
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-xs text-gray-500 uppercase tracking-wider">
            <tr>
              <th className="px-4 py-3 text-left">Nomor HP</th>
              <th className="px-4 py-3 text-left">Pesan</th>
              <th className="px-4 py-3 text-left">Pertama</th>
              <th className="px-4 py-3 text-left">Terakhir</th>
              <th className="px-4 py-3 text-left">Status Leads</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="animate-pulse"><td colSpan={6} className="px-4 py-3"><div className="h-4 bg-gray-100 rounded" /></td></tr>
              ))
            ) : conversations.length === 0 ? (
              <tr><td colSpan={6} className="text-center py-12 text-gray-400">Belum ada percakapan chatbot</td></tr>
            ) : conversations.map((c) => (
              <tr key={c.phone} className="hover:bg-gray-50 cursor-pointer" onClick={() => setSelected(c)}>
                <td className="px-4 py-3 font-medium text-gray-800">{c.phone}</td>
                <td className="px-4 py-3 text-gray-600">{c.message_count}</td>
                <td className="px-4 py-3 text-gray-400 text-xs">{new Date(c.first_chat).toLocaleDateString('id-ID')}</td>
                <td className="px-4 py-3 text-gray-400 text-xs">{new Date(c.last_chat).toLocaleString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}</td>
                <td className="px-4 py-3">
                  {c.lead_id ? (
                    <Link to="/admin/leads" onClick={(e) => e.stopPropagation()}
                      className="px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-700">Leads dibuat</Link>
                  ) : (
                    <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-500">Belum ada leads</span>
                  )}
                </td>
                <td className="px-4 py-3 text-right">
                  {!c.lead_id && (
                    <button onClick={(e) => { e.stopPropagation(); setLeadFor(c.phone) }}
                      className="inline-flex items-center gap-1 px-2 py-1 text-xs bg-blue-50 text-blue-700 rounded hover:bg-blue-100">
                      <UserPlus size={12} /> Buat Leads
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {meta && meta.total_pages > 1 && (
        <div className="flex justify-center gap-2">
          <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)} className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">←</button>
          <span className="px-3 py-1.5 text-sm">Hal {page}/{meta.total_pages}</span>
          <button disabled={page >= meta.total_pages} onClick={() => setPage((p) => p + 1)} className="px-3 py-1.5 border rounded text-sm disabled:opacity-40">→</button>
        </div>
      )}

      {selected && <ConversationDrawer phone={selected.phone} onClose={() => setSelected(null)} />}
      {leadFor && <CreateLeadModal phone={leadFor} onClose={() => setLeadFor(null)} />}
    </div>
  )
}

function ConversationDrawer({ phone, onClose }: { phone: string; onClose: () => void }) {
  const { data: logs, isLoading } = useQuery({
    queryKey: ['chatbot-conversation', phone],
    queryFn: () => getChatbotConversation(phone),
  })
  return (
    <div className="fixed inset-0 bg-black/50 flex justify-end z-50" onClick={onClose}>
      <div className="bg-white w-full max-w-md h-full flex flex-col" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <div className="flex items-center gap-2">
            <MessageSquare size={16} className="text-emerald-600" />
            <span className="font-semibold text-sm">{phone}</span>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600"><X size={18} /></button>
        </div>
        <div className="flex-1 overflow-y-auto p-4 space-y-3 bg-gray-50">
          {isLoading ? (
            <p className="text-center text-gray-400 text-sm">Memuat…</p>
          ) : (logs ?? []).map((m) => (
            <div key={m.id} className={`flex ${m.role === 'assistant' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-[80%] rounded-2xl px-3 py-2 text-sm ${
                m.role === 'assistant' ? 'bg-blue-600 text-white rounded-br-sm' : 'bg-white border text-gray-700 rounded-bl-sm'
              }`}>
                <p className="whitespace-pre-wrap">{m.message}</p>
                <p className={`text-[10px] mt-1 ${m.role === 'assistant' ? 'text-blue-100' : 'text-gray-400'}`}>
                  {new Date(m.created_at).toLocaleString('id-ID', { hour: '2-digit', minute: '2-digit', day: 'numeric', month: 'short' })}
                </p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function CreateLeadModal({ phone, onClose }: { phone: string; onClose: () => void }) {
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [packageId, setPackageId] = useState('')
  const [pax, setPax] = useState(1)

  const { data: pkgData } = useQuery({
    queryKey: ['packages-select'],
    queryFn: () => api.get('/admin/packages?per_page=100').then((r) => r.data),
  })
  const packages: { id: string; name: string }[] = pkgData?.data ?? []

  const mut = useMutation({
    mutationFn: () => createLeadFromChat(phone, { name, package_id: packageId, pax }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chatbot-logs'] })
      toast.success('Leads berhasil dibuat dari percakapan chatbot')
      onClose()
    },
    onError: () => toast.error('Gagal membuat leads'),
  })

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
      <div className="bg-white rounded-xl w-full max-w-sm p-5 space-y-3">
        <h3 className="font-semibold">Buat Leads dari Chat</h3>
        <p className="text-xs text-gray-500">Nomor: <strong>{phone}</strong></p>
        <input className="w-full border rounded-lg px-3 py-2 text-sm" placeholder="Nama lengkap"
          value={name} onChange={(e) => setName(e.target.value)} />
        <select className="w-full border rounded-lg px-3 py-2 text-sm" value={packageId} onChange={(e) => setPackageId(e.target.value)}>
          <option value="">Pilih paket…</option>
          {packages.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
        </select>
        <input type="number" min={1} className="w-full border rounded-lg px-3 py-2 text-sm" placeholder="Pax"
          value={pax} onChange={(e) => setPax(Number(e.target.value))} />
        <div className="flex gap-2 justify-end">
          <button onClick={onClose} className="px-3 py-1.5 border rounded-lg text-sm text-gray-600">Batal</button>
          <button onClick={() => mut.mutate()} disabled={!name || !packageId || mut.isPending}
            className="px-3 py-1.5 bg-emerald-600 text-white rounded-lg text-sm disabled:opacity-50">
            {mut.isPending ? 'Menyimpan…' : 'Buat Leads'}
          </button>
        </div>
      </div>
    </div>
  )
}
