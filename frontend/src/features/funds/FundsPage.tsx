import { forwardRef, type ComponentType, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import {
  AlertTriangle,
  CalendarDays,
  FolderPlus,
  Landmark,
  Loader2,
  Plus,
  RefreshCcw,
  UsersRound,
  WalletCards,
  X,
} from 'lucide-react'
import { createFund, listFunds, type CreateFundPayload } from '@/api/funds'
import Button from '@/components/ui/Button'
import Input from '@/components/ui/Input'
import type { Fund } from '@/types'
import { formatCOP, FUND_STATUS_LABEL, FUND_TYPE_LABEL } from '@/lib/utils'

const createFundSchema = z.object({
  name: z.string().min(3, 'Mínimo 3 caracteres').max(100, 'Máximo 100 caracteres'),
  description: z.string().optional(),
  type: z.enum(['circulo', 'vaca', 'fondo_ahorro']),
  contribution_amount: z.string().regex(/^\d+(\.\d{1,2})?$/, 'Monto inválido'),
  frequency: z.enum(['weekly', 'biweekly', 'monthly']),
  total_periods: z.number().int().min(1, 'Mínimo 1 periodo'),
  start_date: z.string().min(1, 'Selecciona una fecha'),
  min_members: z.number().int().min(1, 'Mínimo 1 miembro'),
  max_members: z.number().int().min(1, 'Mínimo 1 miembro'),
  governance_type: z.enum(['admin_only', 'majority', 'unanimous']),
  voting_deadline_hours: z.number().int().min(1, 'Mínimo 1 hora'),
  penalty_enabled: z.boolean(),
  penalty_type: z.enum(['fixed', 'percentage']),
  penalty_amount: z.string().optional(),
  grace_period_days: z.number().int().min(0),
  payout_order_type: z.enum(['fixed', 'randomized']),
  goal_amount: z.string().optional(),
  goal_description: z.string().optional(),
  distribution_type: z.enum(['goal_reached', 'unanimous_vote']),
  interest_rate: z.string().optional(),
  early_withdrawal_penalty: z.string().optional(),
})

type CreateFundValues = z.infer<typeof createFundSchema>

const defaultValues: CreateFundValues = {
  name: '',
  description: '',
  type: 'fondo_ahorro',
  contribution_amount: '50000',
  frequency: 'monthly',
  total_periods: 12,
  start_date: new Date().toISOString().slice(0, 10),
  min_members: 2,
  max_members: 20,
  governance_type: 'admin_only',
  voting_deadline_hours: 48,
  penalty_enabled: false,
  penalty_type: 'fixed',
  penalty_amount: '0',
  grace_period_days: 3,
  payout_order_type: 'fixed',
  goal_amount: '',
  goal_description: '',
  distribution_type: 'goal_reached',
  interest_rate: '0',
  early_withdrawal_penalty: '0',
}

export default function FundsPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['funds'],
    queryFn: listFunds,
  })
  const funds = Array.isArray(data) ? data : []

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Mis fondos</h1>
          <p className="mt-1 text-sm text-slate-400">Crea fondos, revisa reglas y administra miembros.</p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
            <RefreshCcw className={isFetching ? 'h-4 w-4 animate-spin' : 'h-4 w-4'} />
            Actualizar
          </Button>
          <Button size="sm" onClick={() => setIsCreateOpen(true)}>
            <Plus className="h-4 w-4" />
            Crear fondo
          </Button>
        </div>
      </header>

      {isLoading && (
        <div className="flex min-h-[320px] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-brand-cyan" />
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-danger/30 bg-danger/10 p-5">
          <div className="flex gap-3">
            <AlertTriangle className="mt-0.5 h-5 w-5 text-danger" />
            <div>
              <h2 className="font-semibold text-white">No se pudieron cargar tus fondos</h2>
              <p className="mt-1 text-sm text-slate-400">Intenta refrescar o verifica tu sesión.</p>
            </div>
          </div>
        </div>
      )}

      {!isLoading && !isError && (
        funds.length > 0 ? <FundGrid funds={funds} /> : <EmptyFunds onCreate={() => setIsCreateOpen(true)} />
      )}

      {isCreateOpen && <CreateFundModal onClose={() => setIsCreateOpen(false)} />}
    </div>
  )
}

function FundGrid({ funds }: { funds: Fund[] }) {
  return (
    <div className="grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
      {funds.map((fund) => (
        <Link
          key={fund.id}
          to={`/app/funds/${fund.id}`}
          className="rounded-lg border border-navy-600 bg-navy-800 p-5 shadow-xl shadow-black/10 transition-colors hover:border-brand-cyan/60"
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <p className="truncate text-lg font-semibold text-white">{fund.name}</p>
              <p className="mt-1 text-sm text-slate-500">{FUND_TYPE_LABEL[fund.type]}</p>
            </div>
            <StatusBadge status={fund.status ?? 'draft'} />
          </div>
          {fund.description && <p className="mt-4 line-clamp-2 text-sm text-slate-400">{fund.description}</p>}
          <div className="mt-5 grid grid-cols-2 gap-3 text-sm">
            <InfoTile icon={WalletCards} label="Cuota" value={formatCOP(fund.rules?.contribution_amount ?? '0')} />
            <InfoTile icon={CalendarDays} label="Frecuencia" value={frequencyLabel(fund.rules?.frequency ?? 'monthly')} />
            <InfoTile icon={UsersRound} label="Miembros" value={`${fund.rules?.min_members ?? 0}-${fund.rules?.max_members ?? 0}`} />
            <InfoTile icon={Landmark} label="Periodos" value={String(fund.rules?.total_periods ?? 0)} />
          </div>
        </Link>
      ))}
    </div>
  )
}

function CreateFundModal({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [serverError, setServerError] = useState('')
  const {
    register,
    handleSubmit,
    control,
    formState: { errors, isSubmitting },
  } = useForm<CreateFundValues>({
    resolver: zodResolver(createFundSchema),
    defaultValues,
  })

  const type = useWatch({ control, name: 'type' })
  const penaltyEnabled = useWatch({ control, name: 'penalty_enabled' })

  const mutation = useMutation({
    mutationFn: createFund,
    onSuccess: async (fund) => {
      await queryClient.invalidateQueries({ queryKey: ['funds'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
      navigate(`/app/funds/${fund.id}`)
    },
    onError: (err: unknown) => setServerError(parseApiError(err)),
  })

  function onSubmit(values: CreateFundValues) {
    setServerError('')
    const payload: CreateFundPayload = {
      name: values.name,
      description: values.description,
      type: values.type,
      contribution_amount: values.contribution_amount,
      frequency: values.frequency,
      total_periods: values.total_periods,
      start_date: values.start_date,
      penalty_enabled: values.penalty_enabled,
      penalty_type: values.penalty_enabled ? values.penalty_type : undefined,
      penalty_amount: values.penalty_enabled ? values.penalty_amount || '0' : '0',
      grace_period_days: values.grace_period_days,
      min_members: values.min_members,
      max_members: values.max_members,
      governance_type: values.governance_type,
      voting_deadline_hours: values.voting_deadline_hours,
    }

    if (values.type === 'circulo') payload.payout_order_type = values.payout_order_type
    if (values.type === 'vaca') {
      payload.goal_amount = values.goal_amount || '0'
      payload.goal_description = values.goal_description
      payload.distribution_type = values.distribution_type
    }
    if (values.type === 'fondo_ahorro') {
      payload.interest_rate = values.interest_rate || '0'
      payload.early_withdrawal_penalty = values.early_withdrawal_penalty || '0'
    }

    mutation.mutate(payload)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/60 p-4 backdrop-blur-sm">
      <div className="my-8 w-full max-w-3xl rounded-lg border border-navy-600 bg-navy-800 shadow-2xl">
        <div className="flex items-center justify-between border-b border-navy-600 px-5 py-4">
          <div>
            <h2 className="text-lg font-semibold text-white">Crear fondo</h2>
            <p className="text-sm text-slate-500">Define las reglas iniciales. Se crea en estado borrador.</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-slate-400 hover:bg-navy-700 hover:text-white"
            aria-label="Cerrar"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-5 p-5">
          <div className="grid gap-4 md:grid-cols-2">
            <Input label="Nombre" placeholder="Fondo familiar" error={errors.name?.message} {...register('name')} />
            <SelectField label="Tipo" error={errors.type?.message} {...register('type')}>
              <option value="fondo_ahorro">Fondo de ahorro</option>
              <option value="circulo">Círculo</option>
              <option value="vaca">Vaca</option>
            </SelectField>
          </div>

          <TextAreaField label="Descripción" placeholder="Objetivo o reglas visibles para miembros" {...register('description')} />

          <div className="grid gap-4 md:grid-cols-3">
            <Input label="Cuota" inputMode="decimal" error={errors.contribution_amount?.message} {...register('contribution_amount')} />
            <SelectField label="Frecuencia" error={errors.frequency?.message} {...register('frequency')}>
              <option value="weekly">Semanal</option>
              <option value="biweekly">Quincenal</option>
              <option value="monthly">Mensual</option>
            </SelectField>
            <Input label="Periodos" type="number" min={1} error={errors.total_periods?.message} {...register('total_periods', { valueAsNumber: true })} />
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            <Input label="Fecha inicio" type="date" error={errors.start_date?.message} {...register('start_date')} />
            <Input label="Miembros mín." type="number" min={1} error={errors.min_members?.message} {...register('min_members', { valueAsNumber: true })} />
            <Input label="Miembros máx." type="number" min={1} error={errors.max_members?.message} {...register('max_members', { valueAsNumber: true })} />
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <SelectField label="Gobernanza" error={errors.governance_type?.message} {...register('governance_type')}>
              <option value="admin_only">Solo admin</option>
              <option value="majority">Mayoría</option>
              <option value="unanimous">Unánime</option>
            </SelectField>
            <Input label="Horas para votar" type="number" min={1} error={errors.voting_deadline_hours?.message} {...register('voting_deadline_hours', { valueAsNumber: true })} />
          </div>

          <label className="flex items-center gap-3 rounded-lg border border-navy-600 bg-navy-950/40 px-4 py-3 text-sm text-slate-300">
            <input type="checkbox" className="h-4 w-4 accent-brand-cta" {...register('penalty_enabled')} />
            Aplicar mora por pagos tarde
          </label>

          {penaltyEnabled && (
            <div className="grid gap-4 md:grid-cols-3">
              <SelectField label="Tipo mora" error={errors.penalty_type?.message} {...register('penalty_type')}>
                <option value="fixed">Monto fijo</option>
                <option value="percentage">Porcentaje</option>
              </SelectField>
              <Input label="Valor mora" inputMode="decimal" error={errors.penalty_amount?.message} {...register('penalty_amount')} />
              <Input label="Días de gracia" type="number" min={0} error={errors.grace_period_days?.message} {...register('grace_period_days', { valueAsNumber: true })} />
            </div>
          )}

          {type === 'circulo' && (
            <SelectField label="Orden de pago" error={errors.payout_order_type?.message} {...register('payout_order_type')}>
              <option value="fixed">Definido manualmente</option>
              <option value="randomized">Aleatorio</option>
            </SelectField>
          )}

          {type === 'vaca' && (
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="Meta" inputMode="decimal" error={errors.goal_amount?.message} {...register('goal_amount')} />
              <SelectField label="Distribución" error={errors.distribution_type?.message} {...register('distribution_type')}>
                <option value="goal_reached">Al alcanzar meta</option>
                <option value="unanimous_vote">Por voto unánime</option>
              </SelectField>
              <div className="md:col-span-2">
                <TextAreaField label="Descripción de meta" {...register('goal_description')} />
              </div>
            </div>
          )}

          {type === 'fondo_ahorro' && (
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="Interés (%)" inputMode="decimal" error={errors.interest_rate?.message} {...register('interest_rate')} />
              <Input label="Penalidad retiro temprano (%)" inputMode="decimal" error={errors.early_withdrawal_penalty?.message} {...register('early_withdrawal_penalty')} />
            </div>
          )}

          {serverError && (
            <p className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-sm text-danger">
              {serverError}
            </p>
          )}

          <div className="flex justify-end gap-3 border-t border-navy-600 pt-5">
            <Button type="button" variant="outline" onClick={onClose}>Cancelar</Button>
            <Button type="submit" loading={isSubmitting || mutation.isPending}>
              <FolderPlus className="h-4 w-4" />
              Crear fondo
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

function EmptyFunds({ onCreate }: { onCreate: () => void }) {
  return (
    <section className="rounded-lg border border-dashed border-navy-600 bg-navy-800/70 p-10 text-center">
      <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-lg bg-brand-cyan/15 text-brand-cyan">
        <FolderPlus className="h-6 w-6" />
      </div>
      <h2 className="mt-4 text-lg font-semibold text-white">Aún no tienes fondos</h2>
      <p className="mx-auto mt-2 max-w-md text-sm text-slate-400">
        Crea un fondo de ahorro, una vaca o un círculo para empezar a organizar aportes con tu grupo.
      </p>
      <Button className="mt-5" onClick={onCreate}>
        <Plus className="h-4 w-4" />
        Crear primer fondo
      </Button>
    </section>
  )
}

function InfoTile({ icon: Icon, label, value }: { icon: ComponentType<{ className?: string }>; label: string; value: string }) {
  return (
    <div className="rounded-lg bg-navy-950/40 p-3">
      <Icon className="mb-2 h-4 w-4 text-brand-cyan" />
      <p className="text-xs text-slate-500">{label}</p>
      <p className="mt-1 truncate font-medium text-white">{value}</p>
    </div>
  )
}

function StatusBadge({ status }: { status: Fund['status'] }) {
  const className = {
    draft: 'bg-warning/15 text-warning',
    active: 'bg-success/15 text-success',
    paused: 'bg-slate-500/15 text-slate-300',
    completed: 'bg-brand-cyan/15 text-brand-cyan',
    cancelled: 'bg-danger/15 text-danger',
  }[status] ?? 'bg-slate-500/15 text-slate-300'

  return <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${className}`}>{FUND_STATUS_LABEL[status] ?? status}</span>
}

const SelectField = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement> & { label: string; error?: string }>(
  ({ label, error, children, ...props }, ref) => {
    const id = props.id ?? label.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="flex flex-col gap-1">
        <label htmlFor={id} className="text-sm font-medium text-slate-300">{label}</label>
        <select
          ref={ref}
          id={id}
          {...props}
          className="w-full rounded-lg border border-navy-600 bg-navy-950 px-3.5 py-2.5 text-sm text-white focus:outline-none focus:ring-2 focus:ring-brand-cyan"
        >
          {children}
        </select>
        {error && <p className="text-xs text-danger">{error}</p>}
      </div>
    )
  },
)
SelectField.displayName = 'SelectField'

const TextAreaField = forwardRef<HTMLTextAreaElement, TextareaHTMLAttributes<HTMLTextAreaElement> & { label: string; error?: string }>(
  ({ label, error, ...props }, ref) => {
    const id = props.id ?? label.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="flex flex-col gap-1">
        <label htmlFor={id} className="text-sm font-medium text-slate-300">{label}</label>
        <textarea
          ref={ref}
          id={id}
          rows={3}
          {...props}
          className="w-full rounded-lg border border-navy-600 bg-navy-950 px-3.5 py-2.5 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-cyan"
        />
        {error && <p className="text-xs text-danger">{error}</p>}
      </div>
    )
  },
)
TextAreaField.displayName = 'TextAreaField'

function frequencyLabel(value: string) {
  return { weekly: 'Semanal', biweekly: 'Quincenal', monthly: 'Mensual' }[value] ?? value
}

function parseApiError(err: unknown) {
  const error = (err as { response?: { data?: { error?: { code?: string; message?: string } } } })?.response?.data?.error
  if (error?.code === 'VALIDATION_ERROR') return 'Datos inválidos. Revisa las reglas del fondo.'
  if (error?.message) return error.message
  return 'No se pudo completar la operación.'
}
