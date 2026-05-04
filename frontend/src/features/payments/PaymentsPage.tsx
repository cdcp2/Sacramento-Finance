import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Ban,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  CircleDollarSign,
  Clock,
  Landmark,
  Loader2,
  ReceiptText,
  XCircle,
} from 'lucide-react'
import { getFund, listFundMembers } from '@/api/funds'
import { getAllPayments, getLedger, getMyPayments, payPayment, waivePayment } from '@/api/payments'
import Button from '@/components/ui/Button'
import { useAuthStore } from '@/stores/auth'
import type { FundMember, LedgerEntry, Payment, PaymentStatus } from '@/types'
import { formatCOP, formatDate } from '@/lib/utils'

// ── Status badge ──────────────────────────────────────────────────────────────

const STATUS_CFG: Record<PaymentStatus, { label: string; cls: string; icon: React.ReactNode }> = {
  pending: { label: 'Pendiente', cls: 'bg-warning/15 text-warning border border-warning/30',           icon: <Clock className="h-3 w-3" /> },
  partial: { label: 'Parcial',   cls: 'bg-orange-400/15 text-orange-400 border border-orange-400/30', icon: <Clock className="h-3 w-3" /> },
  paid:    { label: 'Pagado',    cls: 'bg-success/15 text-success border border-success/30',           icon: <CheckCircle2 className="h-3 w-3" /> },
  missed:  { label: 'Vencido',   cls: 'bg-danger/15 text-danger border border-danger/30',             icon: <XCircle className="h-3 w-3" /> },
  waived:  { label: 'Condonado', cls: 'bg-slate-500/15 text-slate-400 border border-slate-500/30',    icon: <Ban className="h-3 w-3" /> },
}

function StatusBadge({ status }: { status: PaymentStatus }) {
  const cfg = STATUS_CFG[status]
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${cfg.cls}`}>
      {cfg.icon} {cfg.label}
    </span>
  )
}

// ── Ledger section ────────────────────────────────────────────────────────────

function LedgerSection({ fundId }: { fundId: string }) {
  const [open, setOpen] = useState(false)
  const { data, isLoading } = useQuery({
    queryKey: ['ledger', fundId],
    queryFn: () => getLedger(fundId),
    enabled: open,
  })

  return (
    <div className="rounded-xl border border-navy-600 bg-navy-700/40 overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between px-5 py-4 text-sm font-medium text-white hover:bg-navy-700/60 transition-colors"
      >
        <span className="flex items-center gap-2">
          <ReceiptText className="h-4 w-4 text-brand-cyan" />
          Libro de movimientos
          {data && (
            <span className="text-brand-cyan font-semibold ml-1">
              — Saldo: {formatCOP(data.balance)}
            </span>
          )}
        </span>
        {open ? <ChevronUp className="h-4 w-4 text-slate-400" /> : <ChevronDown className="h-4 w-4 text-slate-400" />}
      </button>

      {open && (
        <div className="border-t border-navy-600">
          {isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
            </div>
          ) : !data?.entries.length ? (
            <p className="py-6 text-center text-sm text-slate-500">Sin movimientos registrados</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-navy-600 bg-navy-800/40">
                    <th className="text-left px-5 py-2.5 text-xs text-slate-500 font-medium">Fecha</th>
                    <th className="text-left px-5 py-2.5 text-xs text-slate-500 font-medium">Tipo</th>
                    <th className="text-left px-5 py-2.5 text-xs text-slate-500 font-medium">Descripción</th>
                    <th className="text-right px-5 py-2.5 text-xs text-slate-500 font-medium">Monto</th>
                    <th className="text-right px-5 py-2.5 text-xs text-slate-500 font-medium">Saldo fondo</th>
                  </tr>
                </thead>
                <tbody>
                  {data.entries.map((e: LedgerEntry) => (
                    <tr key={e.id} className="border-b border-navy-600/50 hover:bg-navy-700/30">
                      <td className="px-5 py-3 text-slate-400 whitespace-nowrap">{formatDate(e.created_at)}</td>
                      <td className="px-5 py-3 capitalize text-slate-300">{e.type}</td>
                      <td className="px-5 py-3 text-slate-400 max-w-xs truncate">{e.description}</td>
                      <td className={`px-5 py-3 text-right font-medium tabular-nums whitespace-nowrap ${e.direction === 'credit' ? 'text-success' : 'text-danger'}`}>
                        {e.direction === 'credit' ? '+' : '−'}{formatCOP(e.amount)}
                      </td>
                      <td className="px-5 py-3 text-right text-slate-300 tabular-nums whitespace-nowrap">
                        {formatCOP(e.balance_after)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ── Payment row ───────────────────────────────────────────────────────────────

function PaymentRow({
  payment,
  fundId,
  isAdmin,
  memberLabel,
}: {
  payment: Payment
  fundId: string
  isAdmin: boolean
  memberLabel?: string
}) {
  const qc = useQueryClient()
  const isOverdue = payment.status === 'pending' && new Date(payment.due_date) < new Date()
  const canPay   = payment.status === 'pending' || payment.status === 'partial'
  const canWaive = isAdmin && (payment.status === 'pending' || payment.status === 'missed')

  const payMut = useMutation({
    mutationFn: () => payPayment(fundId, payment.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['payments-mine', fundId] })
      qc.invalidateQueries({ queryKey: ['payments-all', fundId] })
      qc.invalidateQueries({ queryKey: ['ledger', fundId] })
    },
  })

  const waiveMut = useMutation({
    mutationFn: () => waivePayment(fundId, payment.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['payments-mine', fundId] })
      qc.invalidateQueries({ queryKey: ['payments-all', fundId] })
    },
  })

  return (
    <tr className="border-b border-navy-600/50 hover:bg-navy-700/30 transition-colors">
      {memberLabel !== undefined && (
        <td className="px-5 py-3.5 text-sm text-slate-300 whitespace-nowrap">{memberLabel}</td>
      )}
      <td className="px-5 py-3.5 text-sm text-slate-300 text-center tabular-nums">
        #{payment.period_number}
      </td>
      <td className={`px-5 py-3.5 text-sm whitespace-nowrap ${isOverdue ? 'text-danger' : 'text-slate-300'}`}>
        {formatDate(payment.due_date)}
        {isOverdue && <span className="ml-1 text-xs font-medium">(vencido)</span>}
      </td>
      <td className="px-5 py-3.5 text-sm text-white text-right tabular-nums whitespace-nowrap font-medium">
        {formatCOP(payment.amount_due)}
      </td>
      <td className="px-5 py-3.5 text-sm text-slate-400 text-right tabular-nums whitespace-nowrap">
        {formatCOP(payment.amount_paid)}
      </td>
      <td className="px-5 py-3.5">
        <StatusBadge status={payment.status} />
      </td>
      <td className="px-5 py-3.5">
        <div className="flex items-center gap-2 justify-end">
          {canPay && (
            <Button size="sm" loading={payMut.isPending} onClick={() => payMut.mutate()}>
              <CircleDollarSign className="h-3.5 w-3.5" />
              Pagar
            </Button>
          )}
          {canWaive && (
            <Button
              size="sm"
              variant="ghost"
              loading={waiveMut.isPending}
              onClick={() => waiveMut.mutate()}
              className="text-slate-400 hover:text-danger"
            >
              <Ban className="h-3.5 w-3.5" />
              Condonar
            </Button>
          )}
          {payment.status === 'paid' && payment.paid_at && (
            <span className="text-xs text-slate-500">{formatDate(payment.paid_at)}</span>
          )}
        </div>
      </td>
    </tr>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

type Tab = 'mine' | 'all'

export default function PaymentsPage() {
  const { fundId } = useParams<{ fundId: string }>()
  const resolvedId = fundId ?? ''
  const currentUser = useAuthStore(s => s.user)
  const [tab, setTab] = useState<Tab>('mine')

  const { data: fund, isLoading: fundLoading } = useQuery({
    queryKey: ['fund', resolvedId],
    queryFn: () => getFund(resolvedId),
    enabled: !!resolvedId,
  })

  const { data: members = [] } = useQuery({
    queryKey: ['members', resolvedId],
    queryFn: () => listFundMembers(resolvedId),
    enabled: !!resolvedId,
  })

  const myMember = members.find((m: FundMember) => m.user_id === currentUser?.id)
  const isAdmin  = myMember?.role === 'admin'

  const { data: myPayments = [], isLoading: mineLoading } = useQuery({
    queryKey: ['payments-mine', resolvedId],
    queryFn: () => getMyPayments(resolvedId),
    enabled: !!resolvedId,
  })

  const { data: allPayments = [], isLoading: allLoading } = useQuery({
    queryKey: ['payments-all', resolvedId],
    queryFn: () => getAllPayments(resolvedId),
    enabled: !!resolvedId && isAdmin && tab === 'all',
  })

  const memberLabel = (memberId: string) => {
    const m = members.find((m: FundMember) => m.id === memberId)
    if (!m) return memberId.substring(0, 8) + '…'
    return m.payout_order ? `Turno #${m.payout_order}` : `M-${m.id.substring(0, 6)}`
  }

  const payments = tab === 'all' ? allPayments : myPayments
  const isLoading = tab === 'mine' ? mineLoading : allLoading

  const summary = {
    pending: payments.filter(p => p.status === 'pending').length,
    paid:    payments.filter(p => p.status === 'paid').length,
    missed:  payments.filter(p => p.status === 'missed').length,
  }

  if (fundLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
      </div>
    )
  }

  return (
    <div className="max-w-5xl mx-auto space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          to={`/app/funds/${resolvedId}`}
          className="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-navy-700 transition-colors"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div>
          <p className="text-xs text-slate-500 uppercase tracking-wide">Pagos</p>
          <h1 className="text-xl font-bold text-white flex items-center gap-2">
            <Landmark className="h-5 w-5 text-brand-cyan" />
            {fund?.name ?? '…'}
          </h1>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        {[
          { label: 'Pendientes', value: summary.pending, color: 'text-warning' },
          { label: 'Pagados',    value: summary.paid,    color: 'text-success' },
          { label: 'Vencidos',   value: summary.missed,  color: 'text-danger'  },
        ].map(({ label, value, color }) => (
          <div key={label} className="rounded-xl bg-navy-700/50 border border-navy-600 px-5 py-4">
            <p className="text-xs text-slate-500 mb-1">{label}</p>
            <p className={`text-2xl font-bold tabular-nums ${color}`}>{value}</p>
          </div>
        ))}
      </div>

      {/* Tabs — admin only */}
      {isAdmin && (
        <div className="flex gap-1 bg-navy-800 p-1 rounded-lg w-fit border border-navy-600">
          {(['mine', 'all'] as Tab[]).map(t => (
            <button
              key={t}
              type="button"
              onClick={() => setTab(t)}
              className={`px-4 py-1.5 rounded-md text-sm font-medium transition-all ${
                tab === t ? 'bg-navy-600 text-white shadow' : 'text-slate-400 hover:text-white'
              }`}
            >
              {t === 'mine' ? 'Mis pagos' : 'Todos los miembros'}
            </button>
          ))}
        </div>
      )}

      {/* Payments table */}
      <div className="rounded-xl border border-navy-600 bg-navy-700/40 overflow-hidden">
        {isLoading ? (
          <div className="flex justify-center py-16">
            <Loader2 className="h-6 w-6 animate-spin text-slate-500" />
          </div>
        ) : payments.length === 0 ? (
          <div className="flex flex-col items-center gap-3 py-16 text-slate-500">
            <CircleDollarSign className="h-8 w-8" />
            {fund?.status === 'draft' ? (
              <>
                <p className="text-sm font-medium text-slate-300">El fondo aún no está activo</p>
                <p className="text-xs text-slate-500 text-center max-w-xs">
                  El calendario de pagos se genera al activar el fondo. Ve al detalle del fondo y actívalo cuando esté listo.
                </p>
                <Link
                  to={`/app/funds/${resolvedId}`}
                  className="mt-1 text-xs text-brand-cyan hover:underline"
                >
                  Ir al detalle del fondo
                </Link>
              </>
            ) : (
              <p className="text-sm">Sin pagos registrados aún</p>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-navy-600 bg-navy-800/40">
                  {tab === 'all' && (
                    <th className="text-left px-5 py-3 text-xs text-slate-500 font-medium">Miembro</th>
                  )}
                  <th className="text-center px-5 py-3 text-xs text-slate-500 font-medium">Período</th>
                  <th className="text-left px-5 py-3 text-xs text-slate-500 font-medium">Vencimiento</th>
                  <th className="text-right px-5 py-3 text-xs text-slate-500 font-medium">Monto</th>
                  <th className="text-right px-5 py-3 text-xs text-slate-500 font-medium">Pagado</th>
                  <th className="text-left px-5 py-3 text-xs text-slate-500 font-medium">Estado</th>
                  <th className="text-right px-5 py-3 text-xs text-slate-500 font-medium">Acción</th>
                </tr>
              </thead>
              <tbody>
                {payments.map(p => (
                  <PaymentRow
                    key={p.id}
                    payment={p}
                    fundId={resolvedId}
                    isAdmin={isAdmin}
                    memberLabel={tab === 'all' ? memberLabel(p.member_id) : undefined}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Ledger */}
      <LedgerSection fundId={resolvedId} />
    </div>
  )
}
