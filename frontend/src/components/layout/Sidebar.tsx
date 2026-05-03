import { NavLink } from 'react-router-dom'
import { Bell, LayoutDashboard, Settings, UsersRound, WalletCards } from 'lucide-react'
import SacramentoLogo from '@/components/brand/SacramentoLogo'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'

const navItems = [
  { to: '/app/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/app/funds', label: 'Mis fondos', icon: WalletCards },
  { to: '/app/notifications', label: 'Notificaciones', icon: Bell },
]

export default function Sidebar() {
  const user = useAuthStore((s) => s.user)
  const initial = user?.full_name?.trim().charAt(0).toUpperCase() ?? 'S'

  return (
    <aside className="flex w-60 shrink-0 flex-col border-r border-navy-600 bg-navy-800 p-4">
      <div className="mb-8">
        <SacramentoLogo size="md" />
      </div>

      <div className="mb-5 flex items-center gap-2 rounded-xl bg-navy-700/70 px-3 py-2 text-sm text-slate-300">
        <UsersRound className="h-4 w-4 text-brand-cyan" />
        <span>Personal</span>
      </div>

      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">Menú</p>
      <nav className="flex flex-col gap-1 text-sm">
        {navItems.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-slate-400 transition-colors hover:bg-navy-700 hover:text-white',
                isActive && 'bg-navy-700 text-brand-cyan',
              )
            }
          >
            <Icon className="h-4 w-4" />
            <span>{label}</span>
          </NavLink>
        ))}
        <NavLink
          to="/app/settings"
          className="flex items-center gap-3 rounded-lg px-3 py-2 text-slate-400 transition-colors hover:bg-navy-700 hover:text-white"
        >
          <Settings className="h-4 w-4" />
          <span>Configuración</span>
        </NavLink>
      </nav>

      <div className="mt-auto flex items-center gap-3 pt-6">
        <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-cta/20 text-sm font-bold text-brand-cta">
          {initial}
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-white">{user?.full_name ?? 'Usuario'}</p>
          <p className="truncate text-xs text-slate-500">{user?.email ?? 'Cuenta personal'}</p>
        </div>
      </div>
    </aside>
  )
}
