import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowDownToLine,
  CheckCircle2,
  CircleDollarSign,
  Loader2,
  RefreshCw,
  Shuffle,
  Target,
  TrendingUp,
} from 'lucide-react'
import {
  accrueInterest,
  getFondoConfig,
  getMyFondoBalance,
  getWithdrawalPreview,
  withdrawFondo,
} from '@/api/fondo'
import {
  assignPayoutOrder,
  closeRound,
  getCirculoConfig,
  listPayouts,
} from '@/api/circulo'
import { distributeVaca, getVacaProgress } from '@/api/vaca'
import Button from '@/components/ui/Button'
import Input from '@/components/ui/Input'
import type { FundMember, FundType, Payout } from '@/types'
import { formatCOP, formatDate } from '@/lib/utils'

// ── Dispatcher ────────────────────────────────────────────────────────────────

export function ProductSection({
  fundId,
  fundType,
  isAdmin,
  members,
}: {
  fundId: string
  fundType: FundType
  isAdmin: boolean
  members: FundMember[]
}) {
  if (fundType === 'vaca') {
    return <VacaPanel fundId={fundId} isAdmin={isAdmin} />
  }
  if (fundType === 'fondo_ahorro') {
    return <FondoPanel fundId={fundId} isAdmin={isAdmin} />
  }
  if (fundType === 'circulo') {
    return <CirculoPanel fundId={fundId} isAdmin={isAdmin} members={members} />
  }
  return null
}

// ── Vaca Panel ────────────────────────────────────────────────────────────────

function VacaPanel({ fundId, isAdmin }: { fundId: string; isAdmin: boolean }) {
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['vaca', fundId],
    queryFn: () => getVacaProgress(fundId),
  })

  const distributeMut = useMutation({
    mutationFn: () => distributeVaca(fundId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vaca', fundId] })
      qc.invalidateQueries({ queryKey: ['ledger', fundId] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })

  if (isLoading) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
      </div>
    )
  }

  if (!data) return null

  const pct = Math.min(100, parseFloat(data.percentage))

  return (
    <div className="space-y-4">
      {data.goal_description && (
        <p className="text-sm text-slate-400 italic">"{data.goal_description}"</p>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <Stat label="Meta" value={formatCOP(data.goal_amount)} icon={Target} color="text-brand-cyan" />
        <Stat label="Acumulado" value={formatCOP(data.current_balance)} icon={CircleDollarSign} color="text-success" />
        <Stat label="Progreso" value={`${pct.toFixed(1)}%`} icon={TrendingUp} color={data.goal_reached ? 'text-success' : 'text-warning'} />
      </div>

      {/* Progress bar */}
      <div className="space-y-1">
        <div className="h-2.5 w-full overflow-hidden rounded-full bg-navy-950/60">
          <div
            className={`h-full rounded-full transition-all ${data.goal_reached ? 'bg-success' : 'bg-brand-cyan'}`}
            style={{ width: `${pct}%` }}
          />
        </div>
        {data.goal_reached && (
          <p className="flex items-center gap-1 text-xs text-success">
            <CheckCircle2 className="h-3.5 w-3.5" />
            ¡Meta alcanzada!
          </p>
        )}
      </div>

      {isAdmin && data.distribution_type === 'goal_reached' && (
        <div className="pt-1">
          <Button
            size="sm"
            loading={distributeMut.isPending}
            onClick={() => distributeMut.mutate()}
            disabled={!data.goal_reached}
          >
            <CircleDollarSign className="h-3.5 w-3.5" />
            Distribuir fondo
          </Button>
          {!data.goal_reached && (
            <p className="mt-1.5 text-xs text-slate-500">
              Disponible cuando se alcance la meta
            </p>
          )}
        </div>
      )}
    </div>
  )
}

// ── Fondo de Ahorro Panel ─────────────────────────────────────────────────────

function FondoPanel({ fundId, isAdmin }: { fundId: string; isAdmin: boolean }) {
  const qc = useQueryClient()
  const [withdrawAmount, setWithdrawAmount] = useState('')
  const [previewOpen, setPreviewOpen] = useState(false)

  const { data: config } = useQuery({
    queryKey: ['fondo-config', fundId],
    queryFn: () => getFondoConfig(fundId),
  })

  const { data: balance, isLoading: balLoading } = useQuery({
    queryKey: ['fondo-balance', fundId],
    queryFn: () => getMyFondoBalance(fundId),
  })

  const { data: preview, isFetching: previewLoading } = useQuery({
    queryKey: ['fondo-preview', fundId, withdrawAmount],
    queryFn: () => getWithdrawalPreview(fundId, withdrawAmount),
    enabled: previewOpen && /^\d+(\.\d{1,2})?$/.test(withdrawAmount) && parseFloat(withdrawAmount) > 0,
  })

  const withdrawMut = useMutation({
    mutationFn: () => withdrawFondo(fundId, withdrawAmount),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fondo-balance', fundId] })
      qc.invalidateQueries({ queryKey: ['ledger', fundId] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      setPreviewOpen(false)
      setWithdrawAmount('')
    },
  })

  const accrueIntMut = useMutation({
    mutationFn: () => accrueInterest(fundId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['fondo-balance', fundId] })
      qc.invalidateQueries({ queryKey: ['ledger', fundId] })
    },
  })

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <Stat
          label="Tu balance"
          value={balLoading ? '…' : formatCOP(balance ?? '0')}
          icon={CircleDollarSign}
          color="text-success"
        />
        <Stat
          label="Tasa de interés"
          value={config ? `${parseFloat(config.interest_rate).toFixed(2)}%` : '…'}
          icon={TrendingUp}
          color="text-brand-cyan"
        />
      </div>

      {/* Withdraw section */}
      <div className="rounded-lg border border-navy-600 bg-navy-950/30 p-4 space-y-3">
        <p className="text-sm font-medium text-white">Retirar fondos</p>
        <div className="flex gap-2">
          <Input
            label=""
            placeholder="Monto a retirar"
            inputMode="decimal"
            value={withdrawAmount}
            onChange={(e) => {
              setWithdrawAmount(e.target.value)
              setPreviewOpen(false)
            }}
            className="flex-1"
          />
          <Button
            size="sm"
            variant="outline"
            className="mt-0 self-end"
            loading={previewLoading}
            disabled={!/^\d+(\.\d{1,2})?$/.test(withdrawAmount) || parseFloat(withdrawAmount) <= 0}
            onClick={() => setPreviewOpen(true)}
          >
            Previsualizar
          </Button>
        </div>

        {previewOpen && preview && (
          <div className="rounded-lg bg-navy-800 p-3 space-y-1.5 text-sm">
            <PreviewRow label="Monto" value={formatCOP(preview.amount)} />
            {preview.is_early && (
              <PreviewRow
                label={`Penalidad (${parseFloat(preview.penalty_rate).toFixed(2)}%)`}
                value={`−${formatCOP(preview.penalty_amount)}`}
                danger
              />
            )}
            <PreviewRow label="Neto a recibir" value={formatCOP(preview.net_amount)} bold />
            <Button
              size="sm"
              loading={withdrawMut.isPending}
              onClick={() => withdrawMut.mutate()}
              className="mt-2 w-full"
            >
              <ArrowDownToLine className="h-3.5 w-3.5" />
              Confirmar retiro
            </Button>
          </div>
        )}
      </div>

      {isAdmin && (
        <Button
          size="sm"
          variant="outline"
          loading={accrueIntMut.isPending}
          onClick={() => accrueIntMut.mutate()}
        >
          <TrendingUp className="h-3.5 w-3.5" />
          Aplicar interés
        </Button>
      )}
    </div>
  )
}

function PreviewRow({ label, value, danger = false, bold = false }: { label: string; value: string; danger?: boolean; bold?: boolean }) {
  return (
    <div className="flex justify-between">
      <span className="text-slate-400">{label}</span>
      <span className={`${danger ? 'text-danger' : bold ? 'text-white font-semibold' : 'text-slate-200'}`}>{value}</span>
    </div>
  )
}

// ── Circulo Panel ─────────────────────────────────────────────────────────────

function CirculoPanel({
  fundId,
  isAdmin,
  members,
}: {
  fundId: string
  isAdmin: boolean
  members: FundMember[]
}) {
  const qc = useQueryClient()
  const [showPayouts, setShowPayouts] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['circulo', fundId],
    queryFn: () => getCirculoConfig(fundId),
  })

  const { data: payouts = [] } = useQuery({
    queryKey: ['circulo-payouts', fundId],
    queryFn: () => listPayouts(fundId),
    enabled: showPayouts,
  })

  const randomizeMut = useMutation({
    mutationFn: () => assignPayoutOrder(fundId, { randomize: true }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['circulo', fundId] }),
  })

  const closeRoundMut = useMutation({
    mutationFn: () => closeRound(fundId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['circulo', fundId] })
      qc.invalidateQueries({ queryKey: ['circulo-payouts', fundId] })
      qc.invalidateQueries({ queryKey: ['ledger', fundId] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })

  if (isLoading) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
      </div>
    )
  }

  if (!data) return null

  const cfg = data.config
  const orderedMembers = [...members]
    .filter((m) => m.payout_order != null)
    .sort((a, b) => (a.payout_order ?? 0) - (b.payout_order ?? 0))
  const unassigned = members.filter((m) => m.payout_order == null)

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-3">
        <Stat label="Ronda actual" value={String(cfg.current_round)} icon={RefreshCw} color="text-brand-cyan" />
        <Stat label="Completadas" value={String(cfg.rounds_completed)} icon={CheckCircle2} color="text-success" />
        <Stat label="Tipo de orden" value={cfg.payout_order_type === 'randomized' ? 'Aleatorio' : 'Manual'} icon={Shuffle} color="text-slate-300" />
      </div>

      {/* Payout order list */}
      <div>
        <p className="mb-2 text-xs font-medium text-slate-400 uppercase tracking-wide">Orden de turnos</p>
        {orderedMembers.length === 0 ? (
          <p className="text-sm text-slate-500">Sin orden asignado aún.</p>
        ) : (
          <div className="divide-y divide-navy-600/50 rounded-lg border border-navy-600 overflow-hidden">
            {orderedMembers.map((m, idx) => {
              const isCurrent = (idx + 1) === cfg.current_round
              return (
                <div
                  key={m.id}
                  className={`flex items-center gap-3 px-4 py-2.5 text-sm ${isCurrent ? 'bg-brand-cyan/10' : ''}`}
                >
                  <span className={`w-6 text-center font-bold tabular-nums ${isCurrent ? 'text-brand-cyan' : 'text-slate-500'}`}>
                    {m.payout_order}
                  </span>
                  <span className={isCurrent ? 'text-white font-medium' : 'text-slate-300'}>
                    {m.role === 'admin' ? 'Admin' : 'Miembro'}
                  </span>
                  <span className="ml-auto text-xs text-slate-500 font-mono">{m.user_id.substring(0, 8)}…</span>
                  {isCurrent && (
                    <span className="rounded-full bg-brand-cyan/20 px-2 py-0.5 text-xs text-brand-cyan">Turno actual</span>
                  )}
                </div>
              )
            })}
          </div>
        )}
        {unassigned.length > 0 && (
          <p className="mt-1.5 text-xs text-warning">{unassigned.length} miembro(s) sin turno asignado</p>
        )}
      </div>

      {isAdmin && (
        <div className="flex flex-wrap gap-2">
          <Button
            size="sm"
            variant="outline"
            loading={randomizeMut.isPending}
            onClick={() => randomizeMut.mutate()}
          >
            <Shuffle className="h-3.5 w-3.5" />
            Asignar orden aleatorio
          </Button>
          <Button
            size="sm"
            loading={closeRoundMut.isPending}
            onClick={() => closeRoundMut.mutate()}
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            Cerrar ronda
          </Button>
        </div>
      )}

      {/* Payouts history toggle */}
      <button
        type="button"
        onClick={() => setShowPayouts((o) => !o)}
        className="text-xs text-brand-cyan hover:underline"
      >
        {showPayouts ? 'Ocultar historial de pagos' : 'Ver historial de pagos'}
      </button>

      {showPayouts && (
        <PayoutsList payouts={payouts} members={members} />
      )}
    </div>
  )
}

function PayoutsList({ payouts, members }: { payouts: Payout[]; members: FundMember[] }) {
  if (payouts.length === 0) {
    return <p className="text-sm text-slate-500">Sin pagos registrados aún.</p>
  }

  const memberLabel = (recipientId: string) => {
    const m = members.find((m) => m.id === recipientId)
    return m ? `Turno #${m.payout_order ?? '?'}` : recipientId.substring(0, 8) + '…'
  }

  return (
    <div className="divide-y divide-navy-600/50 rounded-lg border border-navy-600 overflow-hidden">
      {payouts.map((p) => (
        <div key={p.id} className="flex items-center justify-between gap-4 px-4 py-3 text-sm">
          <div>
            <p className="text-white font-medium">Ronda {p.round_number} → {memberLabel(p.recipient_id)}</p>
            <p className="text-xs text-slate-500">{formatDate(p.scheduled_date)}</p>
          </div>
          <div className="text-right">
            <p className="font-semibold text-success tabular-nums">{formatCOP(p.amount)}</p>
            <p className={`text-xs ${p.status === 'completed' ? 'text-success' : p.status === 'failed' ? 'text-danger' : 'text-slate-400'}`}>
              {p.status === 'completed' ? 'Pagado' : p.status === 'failed' ? 'Fallido' : 'Programado'}
            </p>
          </div>
        </div>
      ))}
    </div>
  )
}

// ── Shared ────────────────────────────────────────────────────────────────────

function Stat({
  label,
  value,
  icon: Icon,
  color,
}: {
  label: string
  value: string
  icon: React.ComponentType<{ className?: string }>
  color: string
}) {
  return (
    <div className="rounded-lg bg-navy-950/40 px-4 py-3">
      <div className="flex items-center gap-1.5 mb-2">
        <Icon className={`h-3.5 w-3.5 ${color}`} />
        <p className="text-xs text-slate-500">{label}</p>
      </div>
      <p className={`text-lg font-bold tabular-nums ${color}`}>{value}</p>
    </div>
  )
}
