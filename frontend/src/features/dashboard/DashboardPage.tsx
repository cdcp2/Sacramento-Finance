import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  AlertTriangle,
  CalendarClock,
  Clock3,
  FolderKanban,
  Landmark,
  Loader2,
  Plus,
  ReceiptText,
  RefreshCcw,
  ShieldCheck,
  UsersRound,
  Vote,
  Wallet,
} from 'lucide-react'
import { getDashboard } from '@/api/dashboard'
import Button from '@/components/ui/Button'
import { useAuthStore } from '@/stores/auth'
import type { DashboardData, DashboardFund, DashboardPayment, DashboardProposal } from '@/types'
import { formatCOP, formatDate, FUND_STATUS_LABEL, FUND_TYPE_LABEL, PAYMENT_STATUS_LABEL } from '@/lib/utils'

const proposalLabel: Record<string, string> = {
  activate_fund: 'Activar fondo',
  change_rules: 'Cambio de reglas',
  waive_payment: 'Exonerar cuota',
  remove_member: 'Retirar miembro',
  cancel_fund: 'Cancelar fondo',
  change_governance: 'Cambiar gobierno',
  distribute_vaca: 'Distribuir vaca',
}

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user)
  const firstName = user?.full_name?.trim().split(/\s+/)[0] ?? 'Usuario'

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['dashboard'],
    queryFn: getDashboard,
  })

  const alerts = useMemo(() => buildAlerts(data), [data])

  if (isLoading) {
    return (
      <div className="flex min-h-[420px] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-brand-cyan" />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <section className="rounded-lg border border-danger/30 bg-danger/10 p-6">
        <div className="flex items-start gap-3">
          <AlertTriangle className="mt-0.5 h-5 w-5 text-danger" />
          <div>
            <h1 className="text-lg font-semibold text-white">No se pudo cargar el dashboard</h1>
            <p className="mt-1 text-sm text-slate-400">Verifica tu sesión o intenta refrescar los datos.</p>
            <Button className="mt-4" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="h-4 w-4" />
              Reintentar
            </Button>
          </div>
        </div>
      </section>
    )
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Hola, {firstName}</h1>
          <p className="mt-1 text-sm text-slate-400">Resumen de tus fondos y pagos pendientes</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCcw className={isFetching ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} />
          Actualizar
        </Button>
      </header>

      <SummaryCards data={data} />

      <div className="flex flex-wrap gap-3">
        <Link to="/app/funds">
          <Button>
            <Plus className="h-4 w-4" />
            Crear fondo
          </Button>
        </Link>
        <Link to="/app/funds">
          <Button variant="outline">
            <Landmark className="h-4 w-4" />
            Solicitar préstamo
          </Button>
        </Link>
        <Link to="/app/funds">
          <Button variant="outline">
            <ReceiptText className="h-4 w-4" />
            Pagar cuota
          </Button>
        </Link>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5 shadow-xl shadow-black/10">
          <SectionTitle title="Próximos vencimientos" icon={CalendarClock} />
          <UpcomingPayments payments={data.upcoming_payments} />
        </section>

        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5 shadow-xl shadow-black/10">
          <SectionTitle title="Alertas" icon={AlertTriangle} />
          <AlertsList alerts={alerts} />
        </section>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5 shadow-xl shadow-black/10">
          <SectionTitle title="Fondos activos" icon={FolderKanban} />
          <FundsList funds={data.funds} />
        </section>

        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5 shadow-xl shadow-black/10">
          <SectionTitle title="Propuestas abiertas" icon={Vote} />
          <ProposalList proposals={data.open_proposals} />
        </section>
      </div>
    </div>
  )
}

function SummaryCards({ data }: { data: DashboardData }) {
  const cards = [
    {
      label: 'Total pendiente',
      value: formatCOP(data.summary.total_pending_amount),
      detail: `${data.summary.pending_payments} cuotas por pagar`,
      icon: Wallet,
      tone: 'text-brand-cyan',
    },
    {
      label: 'Fondos activos',
      value: data.summary.active_funds.toString(),
      detail: `${data.summary.total_funds} fondos en total`,
      icon: UsersRound,
      tone: 'text-brand-cta',
    },
    {
      label: 'Moras activas',
      value: data.summary.overdue_payments.toString(),
      detail: data.summary.overdue_payments > 0 ? 'Requieren atención' : 'Sin cuotas vencidas',
      icon: AlertTriangle,
      tone: data.summary.overdue_payments > 0 ? 'text-warning' : 'text-success',
    },
    {
      label: 'Gobernanza',
      value: data.summary.open_proposals.toString(),
      detail: 'Propuestas abiertas',
      icon: Vote,
      tone: 'text-sky-300',
    },
  ]

  return (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {cards.map(({ label, value, detail, icon: Icon, tone }) => (
        <article key={label} className="rounded-lg border border-navy-600 bg-navy-800 p-5">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-sm text-slate-400">{label}</p>
              <p className="mt-4 text-2xl font-bold text-white">{value}</p>
              <p className={`mt-2 text-xs font-medium ${tone}`}>{detail}</p>
            </div>
            <Icon className={`h-5 w-5 ${tone}`} />
          </div>
        </article>
      ))}
    </div>
  )
}

function UpcomingPayments({ payments }: { payments: DashboardPayment[] }) {
  if (payments.length === 0) {
    return <EmptyState title="No tienes cuotas pendientes" text="Cuando un fondo genere pagos, aparecerán aquí." />
  }

  return (
    <div className="divide-y divide-navy-600/70">
      {payments.map((payment) => (
        <Link
          key={payment.id}
          to={`/app/funds/${payment.fund_id}/payments`}
          className="flex items-center justify-between gap-4 py-4 transition-colors hover:text-white"
        >
          <div className="flex min-w-0 items-center gap-3">
            <div className={payment.is_overdue ? 'dashboard-entry-icon danger' : 'dashboard-entry-icon warning'}>
              <Clock3 className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-white">{payment.fund_name}</p>
              <p className="text-xs text-slate-500">
                Cuota {payment.period_number} · {formatDate(payment.due_date)}
              </p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-sm font-semibold text-white">{formatCOP(payment.amount_due)}</p>
            <p className={payment.is_overdue ? 'text-xs text-danger' : 'text-xs text-warning'}>
              {payment.is_overdue ? 'En mora' : PAYMENT_STATUS_LABEL[payment.status]}
            </p>
          </div>
        </Link>
      ))}
    </div>
  )
}

function AlertsList({ alerts }: { alerts: Array<{ tone: string; text: string }> }) {
  if (alerts.length === 0) {
    return <EmptyState title="Sin alertas" text="Todo está al día en tu cuenta." />
  }

  return (
    <div className="space-y-3">
      {alerts.map((alert) => (
        <div key={alert.text} className={`rounded-lg px-4 py-3 text-sm ${alert.tone}`}>
          {alert.text}
        </div>
      ))}
    </div>
  )
}

function FundsList({ funds }: { funds: DashboardFund[] }) {
  if (funds.length === 0) {
    return <EmptyState title="Aún no tienes fondos" text="Crea un fondo o únete a uno para comenzar." />
  }

  return (
    <div className="grid gap-3 md:grid-cols-2">
      {funds.map((fund) => (
        <Link
          key={fund.id}
          to={`/app/funds/${fund.id}`}
          className="rounded-lg border border-navy-600 bg-navy-950/40 p-4 transition-colors hover:border-brand-cyan/60"
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <p className="truncate font-semibold text-white">{fund.name}</p>
              <p className="mt-1 text-xs text-slate-500">
                {FUND_TYPE_LABEL[fund.type]} · {FUND_STATUS_LABEL[fund.status]}
              </p>
            </div>
            {fund.role === 'admin' && <ShieldCheck className="h-4 w-4 shrink-0 text-brand-cyan" />}
          </div>
          <div className="mt-4 grid grid-cols-3 gap-2 text-xs">
            <MiniStat label="Pagos" value={fund.pending_payments} />
            <MiniStat label="Moras" value={fund.overdue_payments} />
            <MiniStat label="Votos" value={fund.open_proposals} />
          </div>
        </Link>
      ))}
    </div>
  )
}

function ProposalList({ proposals }: { proposals: DashboardProposal[] }) {
  if (proposals.length === 0) {
    return <EmptyState title="Sin propuestas abiertas" text="Las decisiones pendientes aparecerán aquí." />
  }

  return (
    <div className="space-y-3">
      {proposals.map((proposal) => (
        <Link
          key={proposal.id}
          to={`/app/funds/${proposal.fund_id}`}
          className="block rounded-lg border border-navy-600 bg-navy-950/40 p-4 transition-colors hover:border-brand-cyan/60"
        >
          <p className="font-medium text-white">{proposalLabel[proposal.type] ?? proposal.type}</p>
          <p className="mt-1 text-xs text-slate-500">{proposal.fund_name}</p>
          <div className="mt-3 flex items-center justify-between text-xs">
            <span className="text-brand-cta">{proposal.votes_for} a favor</span>
            <span className="text-danger">{proposal.votes_against} en contra</span>
            <span className="text-slate-400">Quórum {proposal.quorum_needed}</span>
          </div>
        </Link>
      ))}
    </div>
  )
}

function SectionTitle({ title, icon: Icon }: { title: string; icon: React.ComponentType<{ className?: string }> }) {
  return (
    <div className="mb-4 flex items-center gap-2">
      <Icon className="h-4 w-4 text-brand-cyan" />
      <h2 className="font-semibold text-white">{title}</h2>
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md bg-navy-700/60 px-3 py-2">
      <p className="font-semibold text-white">{value}</p>
      <p className="mt-0.5 text-slate-500">{label}</p>
    </div>
  )
}

function EmptyState({ title, text }: { title: string; text: string }) {
  return (
    <div className="rounded-lg border border-dashed border-navy-600 bg-navy-950/30 p-6 text-center">
      <p className="font-medium text-slate-200">{title}</p>
      <p className="mt-1 text-sm text-slate-500">{text}</p>
    </div>
  )
}

function buildAlerts(data?: DashboardData) {
  if (!data) return []

  const alerts: Array<{ tone: string; text: string }> = []
  if (data.summary.overdue_payments > 0) {
    alerts.push({
      tone: 'bg-danger/20 text-red-300',
      text: `${data.summary.overdue_payments} cuota(s) en mora por revisar`,
    })
  }
  if (data.summary.pending_payments > 0) {
    alerts.push({
      tone: 'bg-warning/20 text-warning',
      text: `${data.summary.pending_payments} pago(s) pendientes por ${formatCOP(data.summary.total_pending_amount)}`,
    })
  }
  if (data.summary.open_proposals > 0) {
    alerts.push({
      tone: 'bg-brand-cyan/15 text-brand-cyan',
      text: `${data.summary.open_proposals} propuesta(s) esperando votos`,
    })
  }
  if (data.summary.unread_notifications > 0) {
    alerts.push({
      tone: 'bg-brand-cta/15 text-brand-cta',
      text: `${data.summary.unread_notifications} notificación(es) sin leer`,
    })
  }

  return alerts
}
