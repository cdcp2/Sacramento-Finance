import { Outlet } from 'react-router-dom'
import { Bell } from 'lucide-react'
import Sidebar from './Sidebar'
import SacramentoLogo from '@/components/brand/SacramentoLogo'
import { useAuthStore } from '@/stores/auth'

export default function AppLayout() {
  const user = useAuthStore((s) => s.user)

  return (
    <div className="flex h-screen overflow-hidden bg-navy-900">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-14 shrink-0 items-center justify-between border-b border-navy-600 bg-navy-950/60 px-6">
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-slate-200">
              {user?.full_name ?? 'Sacramento Finance'}
            </p>
            <p className="text-xs text-slate-500">Cuenta personal</p>
          </div>

          <div className="flex items-center gap-4">
            <button
              type="button"
              className="relative inline-flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-navy-700 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-cyan"
              aria-label="Notificaciones"
            >
              <Bell className="h-4 w-4" />
              <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-danger" />
            </button>
            <SacramentoLogo size="sm" showText={false} />
          </div>
        </header>

        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
