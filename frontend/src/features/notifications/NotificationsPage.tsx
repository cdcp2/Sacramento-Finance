import { useMutation, useInfiniteQuery, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  Ban,
  Bell,
  BellOff,
  Check,
  CheckCircle2,
  ChevronDown,
  CircleDollarSign,
  FileText,
  Loader2,
  Mail,
  RotateCcw,
  ShieldCheck,
  TrendingUp,
  Trophy,
  UserPlus,
  X,
} from 'lucide-react'
import { acceptInvitation, listMyInvitations, rejectInvitation } from '@/api/funds'
import { getNotifications, markAllRead, markRead } from '@/api/notifications'
import Button from '@/components/ui/Button'
import type { FundInvitation, Notification, NotificationType } from '@/types'
import { formatDate, formatRelative } from '@/lib/utils'

// ── Notification icon & color by type ────────────────────────────────────────

const NOTIF_CFG: Record<NotificationType, { icon: React.ReactNode; color: string }> = {
  payment_confirmed: { icon: <CheckCircle2 className="h-4 w-4" />, color: 'text-success bg-success/15' },
  payment_waived:    { icon: <Ban className="h-4 w-4" />,          color: 'text-slate-400 bg-slate-500/15' },
  payment_overdue:   { icon: <AlertTriangle className="h-4 w-4" />,color: 'text-danger bg-danger/15' },
  payout_received:   { icon: <CircleDollarSign className="h-4 w-4" />, color: 'text-success bg-success/15' },
  round_closed:      { icon: <RotateCcw className="h-4 w-4" />,    color: 'text-brand-cyan bg-brand-cyan/15' },
  withdrawal:        { icon: <TrendingUp className="h-4 w-4 rotate-180" />, color: 'text-warning bg-warning/15' },
  interest_accrued:  { icon: <TrendingUp className="h-4 w-4" />,   color: 'text-brand-blue bg-brand-blue/15' },
  goal_reached:      { icon: <Trophy className="h-4 w-4" />,        color: 'text-yellow-400 bg-yellow-400/15' },
  member_joined:     { icon: <UserPlus className="h-4 w-4" />,      color: 'text-brand-cyan bg-brand-cyan/15' },
  member_left:       { icon: <UserPlus className="h-4 w-4 rotate-180" />, color: 'text-slate-400 bg-slate-500/15' },
  fund_invitation:   { icon: <Mail className="h-4 w-4" />,          color: 'text-brand-cyan bg-brand-cyan/15' },
  proposal_created:  { icon: <FileText className="h-4 w-4" />,      color: 'text-brand-blue bg-brand-blue/15' },
  proposal_resolved: { icon: <ShieldCheck className="h-4 w-4" />,   color: 'text-success bg-success/15' },
}

// ── Single notification item ──────────────────────────────────────────────────

function NotifItem({ notif }: { notif: Notification }) {
  const qc = useQueryClient()
  const cfg = NOTIF_CFG[notif.type] ?? { icon: <Bell className="h-4 w-4" />, color: 'text-slate-400 bg-slate-500/15' }

  const readMut = useMutation({
    mutationFn: () => markRead(notif.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] })
      qc.invalidateQueries({ queryKey: ['notifications-unread'] })
    },
  })

  return (
    <div className={`flex items-start gap-4 py-4 transition-colors ${!notif.is_read ? 'opacity-100' : 'opacity-60'}`}>
      {/* Unread dot */}
      <div className="mt-1 flex-shrink-0 relative">
        <div className={`flex h-8 w-8 items-center justify-center rounded-full ${cfg.color}`}>
          {cfg.icon}
        </div>
        {!notif.is_read && (
          <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-brand-cyan border-2 border-navy-800" />
        )}
      </div>

      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-white">{notif.title}</p>
        <p className="mt-0.5 text-sm text-slate-400">{notif.body}</p>
        <p className="mt-1 text-xs text-slate-500">{formatRelative(notif.created_at)}</p>
      </div>

      {!notif.is_read && (
        <button
          type="button"
          onClick={() => readMut.mutate()}
          disabled={readMut.isPending}
          className="mt-1 flex-shrink-0 text-xs text-slate-500 hover:text-brand-cyan transition-colors"
          title="Marcar como leída"
        >
          <Check className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}

// ── Invitations section ───────────────────────────────────────────────────────

function InvitationItem({ invitation }: { invitation: FundInvitation }) {
  const qc = useQueryClient()

  const acceptMut = useMutation({
    mutationFn: () => acceptInvitation(invitation.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['my-invitations'] })
      qc.invalidateQueries({ queryKey: ['funds'] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
  const rejectMut = useMutation({
    mutationFn: () => rejectInvitation(invitation.id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['my-invitations'] }),
  })

  const statusCls: Record<string, string> = {
    pending:   'bg-warning/15 text-warning',
    accepted:  'bg-success/15 text-success',
    rejected:  'bg-danger/15 text-danger',
    cancelled: 'bg-slate-500/15 text-slate-300',
    expired:   'bg-slate-500/15 text-slate-300',
  }
  const statusLabel: Record<string, string> = {
    pending: 'Pendiente', accepted: 'Aceptada', rejected: 'Rechazada',
    cancelled: 'Cancelada', expired: 'Expirada',
  }

  return (
    <div className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-brand-cyan/15 text-brand-cyan">
          <Mail className="h-4 w-4" />
        </div>
        <div>
          <p className="text-sm font-medium text-white">Invitación a fondo</p>
          <p className="text-xs text-slate-500">Expira {formatDate(invitation.expires_at)}</p>
          {invitation.message && (
            <p className="mt-1 text-xs text-slate-400 italic">"{invitation.message}"</p>
          )}
        </div>
      </div>

      {invitation.status === 'pending' ? (
        <div className="flex gap-2 sm:flex-shrink-0">
          <Button size="sm" onClick={() => acceptMut.mutate()} loading={acceptMut.isPending}>
            <Check className="h-3.5 w-3.5" /> Aceptar
          </Button>
          <Button size="sm" variant="outline" onClick={() => rejectMut.mutate()} loading={rejectMut.isPending}>
            <X className="h-3.5 w-3.5" /> Rechazar
          </Button>
        </div>
      ) : (
        <span className={`self-start rounded-full px-2.5 py-1 text-xs font-medium sm:self-auto ${statusCls[invitation.status]}`}>
          {statusLabel[invitation.status]}
        </span>
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

const PAGE_SIZE = 20

export default function NotificationsPage() {
  const qc = useQueryClient()

  const { data: invitations = [], isLoading: invLoad } = useQuery({
    queryKey: ['my-invitations'],
    queryFn: listMyInvitations,
    select: d => (Array.isArray(d) ? d : []),
  })

  const {
    data: notifPages,
    isLoading: notifLoad,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
  } = useInfiniteQuery({
    queryKey: ['notifications'],
    queryFn: ({ pageParam }) => getNotifications(PAGE_SIZE, pageParam as number),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) =>
      lastPage.notifications.length === PAGE_SIZE ? allPages.length * PAGE_SIZE : undefined,
  })

  const notifications = notifPages?.pages.flatMap(p => p.notifications) ?? []
  const unreadCount   = notifPages?.pages[0]?.unread_count ?? 0

  const markAllMut = useMutation({
    mutationFn: markAllRead,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications'] })
      qc.invalidateQueries({ queryKey: ['notifications-unread'] })
    },
  })

  const pendingInvites = invitations.filter(i => i.status === 'pending')
  const otherInvites   = invitations.filter(i => i.status !== 'pending')

  return (
    <div className="max-w-2xl mx-auto space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Notificaciones</h1>
          <p className="mt-1 text-sm text-slate-400">
            {unreadCount > 0 ? `${unreadCount} sin leer` : 'Todo al día'}
          </p>
        </div>
        {unreadCount > 0 && (
          <Button
            size="sm"
            variant="ghost"
            loading={markAllMut.isPending}
            onClick={() => markAllMut.mutate()}
          >
            <Check className="h-4 w-4" />
            Marcar todo como leído
          </Button>
        )}
      </div>

      {/* Pending invitations — action required */}
      {(invLoad || pendingInvites.length > 0) && (
        <section className="rounded-xl border border-warning/30 bg-warning/5 px-5 overflow-hidden">
          <div className="flex items-center gap-2 py-3 border-b border-warning/20">
            <Mail className="h-4 w-4 text-warning" />
            <h2 className="text-sm font-semibold text-warning">Invitaciones pendientes</h2>
            {pendingInvites.length > 0 && (
              <span className="ml-auto rounded-full bg-warning/20 px-2 py-0.5 text-xs font-medium text-warning">
                {pendingInvites.length}
              </span>
            )}
          </div>
          {invLoad ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
            </div>
          ) : pendingInvites.length === 0 ? (
            <p className="py-4 text-sm text-slate-500">Sin invitaciones pendientes</p>
          ) : (
            <div className="divide-y divide-warning/10">
              {pendingInvites.map(i => <InvitationItem key={i.id} invitation={i} />)}
            </div>
          )}
        </section>
      )}

      {/* Activity feed */}
      <section className="rounded-xl border border-navy-600 bg-navy-700/40 overflow-hidden">
        <div className="flex items-center gap-2 px-5 py-3 border-b border-navy-600">
          <Bell className="h-4 w-4 text-brand-cyan" />
          <h2 className="text-sm font-semibold text-white">Actividad reciente</h2>
        </div>

        {notifLoad ? (
          <div className="flex justify-center py-12">
            <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
          </div>
        ) : notifications.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-12 text-slate-500">
            <BellOff className="h-7 w-7" />
            <p className="text-sm">Sin actividad reciente</p>
          </div>
        ) : (
          <>
            <div className="divide-y divide-navy-600/50 px-5">
              {notifications.map(n => <NotifItem key={n.id} notif={n} />)}
            </div>
            {(hasNextPage || isFetchingNextPage) && (
              <div className="border-t border-navy-600 px-5 py-3 text-center">
                <Button
                  size="sm"
                  variant="ghost"
                  loading={isFetchingNextPage}
                  onClick={() => fetchNextPage()}
                >
                  <ChevronDown className="h-4 w-4" />
                  Cargar más
                </Button>
              </div>
            )}
          </>
        )}
      </section>

      {/* Past invitations (accepted/rejected) — collapsed count */}
      {otherInvites.length > 0 && (
        <section className="rounded-xl border border-navy-600 bg-navy-700/40 overflow-hidden">
          <div className="flex items-center gap-2 px-5 py-3 border-b border-navy-600">
            <Mail className="h-4 w-4 text-slate-500" />
            <h2 className="text-sm font-semibold text-slate-400">Invitaciones anteriores</h2>
          </div>
          <div className="divide-y divide-navy-600/50 px-5">
            {otherInvites.map(i => <InvitationItem key={i.id} invitation={i} />)}
          </div>
        </section>
      )}
    </div>
  )
}
