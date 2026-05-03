export default function Sidebar() {
  return (
    <aside className="w-56 bg-navy-800 border-r border-navy-600 flex flex-col p-4">
      <span className="text-brand-cyan font-bold text-lg mb-8">Sacramento</span>
      <nav className="flex flex-col gap-2 text-sm text-slate-400">
        <a href="/app/dashboard" className="hover:text-white">Dashboard</a>
        <a href="/app/funds" className="hover:text-white">Mis fondos</a>
        <a href="/app/notifications" className="hover:text-white">Notificaciones</a>
      </nav>
    </aside>
  )
}
