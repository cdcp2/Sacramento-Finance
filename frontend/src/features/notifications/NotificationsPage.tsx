import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Loader2, Mail, X } from 'lucide-react'
import { acceptInvitation, listMyInvitations, rejectInvitation } from '@/api/funds'
import Button from '@/components/ui/Button'
import type { FundInvitation } from '@/types'
import { formatDate } from '@/lib/utils'

export default function NotificationsPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['my-invitations'],
    queryFn: listMyInvitations,
  })
  const invitations = Array.isArray(data) ? data : []

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-white">Notificaciones</h1>
        <p className="mt-1 text-sm text-slate-400">Revisa invitaciones pendientes y decisiones de fondos.</p>
      </header>

      <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
        <div className="mb-4 flex items-center gap-2">
          <Mail className="h-4 w-4 text-brand-cyan" />
          <h2 className="font-semibold text-white">Invitaciones a fondos</h2>
        </div>

        {isLoading ? (
          <div className="flex min-h-40 items-center justify-center">
            <Loader2 className="h-7 w-7 animate-spin text-brand-cyan" />
          </div>
        ) : (
          <InvitationList invitations={invitations} />
        )}
      </section>
    </div>
  )
}

function InvitationList({ invitations }: { invitations?: FundInvitation[] | null }) {
  const safeInvitations = Array.isArray(invitations) ? invitations : []

  if (safeInvitations.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-navy-600 bg-navy-950/30 p-6 text-center">
        <p className="font-medium text-slate-200">No tienes invitaciones</p>
        <p className="mt-1 text-sm text-slate-500">Cuando alguien te invite a un fondo aparecerá aquí.</p>
      </div>
    )
  }

  return (
    <div className="divide-y divide-navy-600/70">
      {safeInvitations.map((invitation) => (
        <InvitationItem key={invitation.id} invitation={invitation} />
      ))}
    </div>
  )
}

function InvitationItem({ invitation }: { invitation: FundInvitation }) {
  const queryClient = useQueryClient()
  const acceptMutation = useMutation({
    mutationFn: () => acceptInvitation(invitation.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['my-invitations'] })
      await queryClient.invalidateQueries({ queryKey: ['funds'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
  const rejectMutation = useMutation({
    mutationFn: () => rejectInvitation(invitation.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['my-invitations'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
  const isPending = invitation.status === 'pending'

  return (
    <div className="flex flex-col gap-4 py-4 md:flex-row md:items-center md:justify-between">
      <div className="min-w-0">
        <p className="font-medium text-white">Invitación a fondo</p>
        <p className="mt-1 truncate text-xs text-slate-500">Fondo: {invitation.fund_id}</p>
        <p className="mt-1 text-xs text-slate-500">Expira {formatDate(invitation.expires_at)}</p>
      </div>

      {isPending ? (
        <div className="flex gap-2">
          <Button size="sm" onClick={() => acceptMutation.mutate()} loading={acceptMutation.isPending}>
            <Check className="h-4 w-4" />
            Aceptar
          </Button>
          <Button size="sm" variant="outline" onClick={() => rejectMutation.mutate()} disabled={rejectMutation.isPending}>
            <X className="h-4 w-4" />
            Rechazar
          </Button>
        </div>
      ) : (
        <InvitationBadge status={invitation.status} />
      )}
    </div>
  )
}

function InvitationBadge({ status }: { status: FundInvitation['status'] }) {
  const className = {
    pending: 'bg-warning/15 text-warning',
    accepted: 'bg-success/15 text-success',
    rejected: 'bg-danger/15 text-danger',
    cancelled: 'bg-slate-500/15 text-slate-300',
    expired: 'bg-slate-500/15 text-slate-300',
  }[status]
  const label = {
    pending: 'Pendiente',
    accepted: 'Aceptada',
    rejected: 'Rechazada',
    cancelled: 'Cancelada',
    expired: 'Expirada',
  }[status]

  return <span className={`self-start rounded-full px-2.5 py-1 text-xs font-medium ${className}`}>{label}</span>
}
