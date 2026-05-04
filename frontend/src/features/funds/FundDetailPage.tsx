import { forwardRef, useState, type ComponentType, type SelectHTMLAttributes } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import {
  AlertTriangle,
  ArrowLeft,
  CalendarDays,
  CheckCircle2,
  Clock3,
  FileText,
  Landmark,
  Loader2,
  Mail,
  Pencil,
  ReceiptText,
  ShieldCheck,
  ThumbsDown,
  ThumbsUp,
  UserPlus,
  UsersRound,
  WalletCards,
  X,
} from 'lucide-react'
import {
  activateFund,
  createFundInvitation,
  getFund,
  listFundInvitations,
  listFundMembers,
  updateFundRules,
  type CreateInvitationPayload,
} from '@/api/funds'
import { getAllPayments } from '@/api/payments'
import { castVote, createProposal, getProposal, listProposals } from '@/api/proposals'
import { ProductSection } from './ProductPanel'
import Button from '@/components/ui/Button'
import Input from '@/components/ui/Input'
import { useAuthStore } from '@/stores/auth'
import type { Fund, FundInvitation, FundMember, Payment, Proposal, ProposalStatus, ProposalType, VoteChoice } from '@/types'
import { formatCOP, formatDate, FUND_STATUS_LABEL, FUND_TYPE_LABEL } from '@/lib/utils'

const addMemberSchema = z
  .object({
    method: z.enum(['email', 'document', 'user_id']),
    email: z.string().optional(),
    document_type: z.enum(['cedula_ciudadania', 'cedula_extranjeria', 'pasaporte']),
    document_number: z.string().optional(),
    user_id: z.string().optional(),
  })
  .superRefine((value, ctx) => {
    if (value.method === 'email' && !z.string().email().safeParse(value.email).success) {
      ctx.addIssue({ code: 'custom', path: ['email'], message: 'Correo inválido' })
    }
    if (value.method === 'document' && !value.document_number) {
      ctx.addIssue({ code: 'custom', path: ['document_number'], message: 'Documento requerido' })
    }
    if (value.method === 'user_id' && !z.string().uuid().safeParse(value.user_id).success) {
      ctx.addIssue({ code: 'custom', path: ['user_id'], message: 'UUID inválido' })
    }
  })

type AddMemberValues = z.infer<typeof addMemberSchema>

export default function FundDetailPage() {
  const { fundId } = useParams()
  const resolvedFundId = fundId ?? ''

  const fundQuery = useQuery({
    queryKey: ['fund', resolvedFundId],
    queryFn: () => getFund(resolvedFundId),
    enabled: Boolean(resolvedFundId),
  })
  const membersQuery = useQuery({
    queryKey: ['fund-members', resolvedFundId],
    queryFn: () => listFundMembers(resolvedFundId),
    enabled: Boolean(resolvedFundId),
  })

  if (!resolvedFundId) {
    return <InvalidFund />
  }

  if (fundQuery.isLoading || membersQuery.isLoading) {
    return (
      <div className="flex min-h-[420px] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-brand-cyan" />
      </div>
    )
  }

  if (fundQuery.isError || !fundQuery.data) {
    return (
      <section className="rounded-lg border border-danger/30 bg-danger/10 p-6">
        <div className="flex gap-3">
          <AlertTriangle className="mt-0.5 h-5 w-5 text-danger" />
          <div>
            <h1 className="font-semibold text-white">No se pudo cargar el fondo</h1>
            <p className="mt-1 text-sm text-slate-400">Verifica que seas miembro o vuelve al listado.</p>
            <Link to="/app/funds">
              <Button className="mt-4" size="sm" variant="outline">
                <ArrowLeft className="h-4 w-4" />
                Volver
              </Button>
            </Link>
          </div>
        </div>
      </section>
    )
  }

  return <FundDetail fund={fundQuery.data} members={membersQuery.data ?? []} />
}

function FundDetail({ fund, members }: { fund: Fund; members: FundMember[] }) {
  const queryClient = useQueryClient()
  const user = useAuthStore((s) => s.user)
  const [serverError, setServerError] = useState('')
  const [editRulesOpen, setEditRulesOpen] = useState(false)
  const currentMember = members.find((member) => member.user_id === user?.id)
  const isAdmin = currentMember?.role === 'admin'
  const canEditRules = isAdmin && fund.status === 'draft' && fund.rules?.governance_type === 'admin_only'

  const activateMutation = useMutation({
    mutationFn: () => activateFund(fund.id),
    onSuccess: async () => {
      setServerError('')
      await queryClient.invalidateQueries({ queryKey: ['fund', fund.id] })
      await queryClient.invalidateQueries({ queryKey: ['funds'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
    onError: (err: unknown) => setServerError(parseApiError(err)),
  })

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <Link to="/app/funds" className="mb-4 inline-flex items-center gap-2 text-sm text-slate-400 hover:text-white">
            <ArrowLeft className="h-4 w-4" />
            Volver a fondos
          </Link>
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{fund.name}</h1>
            <StatusBadge status={fund.status} />
            {isAdmin && (
              <span className="inline-flex items-center gap-1 rounded-full bg-brand-cyan/15 px-2.5 py-1 text-xs font-medium text-brand-cyan">
                <ShieldCheck className="h-3.5 w-3.5" />
                Admin
              </span>
            )}
          </div>
          <p className="mt-2 text-sm text-slate-400">{fund.description || FUND_TYPE_LABEL[fund.type]}</p>
        </div>

        <div className="flex flex-wrap gap-3">
          <Link to={`/app/funds/${fund.id}/payments`}>
            <Button variant="outline" size="sm">
              <ReceiptText className="h-4 w-4" />
              Pagos
            </Button>
          </Link>
          {canEditRules && (
            <Button variant="outline" size="sm" onClick={() => setEditRulesOpen(true)}>
              <Pencil className="h-4 w-4" />
              Editar reglas
            </Button>
          )}
          {isAdmin && fund.status === 'draft' && (
            <Button size="sm" onClick={() => activateMutation.mutate()} loading={activateMutation.isPending}>
              <CheckCircle2 className="h-4 w-4" />
              Activar fondo
            </Button>
          )}
        </div>

        {editRulesOpen && (
          <EditRulesModal
            fund={fund}
            onClose={() => setEditRulesOpen(false)}
            onSaved={(updated) => {
              queryClient.setQueryData(['fund', fund.id], updated)
              queryClient.invalidateQueries({ queryKey: ['funds'] })
              setEditRulesOpen(false)
            }}
          />
        )}
      </header>

      {serverError && (
        <p className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-sm text-danger">
          {serverError}
        </p>
      )}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Metric icon={WalletCards} label="Cuota" value={formatCOP(fund.rules?.contribution_amount ?? '0')} />
        <Metric icon={CalendarDays} label="Inicio" value={fund.rules?.start_date ? formatDate(fund.rules.start_date) : 'Sin fecha'} />
        <Metric icon={Clock3} label="Frecuencia" value={frequencyLabel(fund.rules?.frequency ?? 'monthly')} />
        <Metric icon={UsersRound} label="Miembros" value={`${members.length}/${fund.rules?.max_members ?? 0}`} />
      </div>

      <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
        <SectionTitle icon={WalletCards} title={PRODUCT_TITLE[fund.type] ?? 'Producto'} />
        <ProductSection fundId={fund.id} fundType={fund.type} isAdmin={isAdmin} members={members} />
      </section>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_420px]">
        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
          <SectionTitle icon={Landmark} title="Reglas del fondo" />
          <RulesGrid fund={fund} />
        </section>

        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
          <SectionTitle icon={UserPlus} title="Invitar miembro" />
          {isAdmin ? <InviteMemberForm fundId={fund.id} /> : (
            <p className="text-sm text-slate-400">Solo administradores pueden invitar miembros.</p>
          )}
        </section>
      </div>

      <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
        <SectionTitle icon={UsersRound} title="Miembros" />
        <MembersList members={members} creatorId={fund.creator_id} />
      </section>

      <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
        <SectionTitle icon={FileText} title="Gobernanza" />
        <GovernanceSection fundId={fund.id} isAdmin={isAdmin} fundType={fund.type} members={members} />
      </section>

      {isAdmin && (
        <section className="rounded-lg border border-navy-600 bg-navy-800 p-5">
          <SectionTitle icon={Mail} title="Invitaciones" />
          <AdminInvitationList fundId={fund.id} />
        </section>
      )}
    </div>
  )
}

function AdminInvitationList({ fundId }: { fundId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['fund-invitations', fundId],
    queryFn: () => listFundInvitations(fundId),
  })
  const invitations = Array.isArray(data) ? data : []

  if (isLoading) {
    return (
      <div className="flex min-h-24 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-brand-cyan" />
      </div>
    )
  }

  return <InvitationList invitations={invitations} />
}

function InviteMemberForm({ fundId }: { fundId: string }) {
  const queryClient = useQueryClient()
  const [serverError, setServerError] = useState('')
  const [success, setSuccess] = useState('')
  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors },
  } = useForm<AddMemberValues>({
    resolver: zodResolver(addMemberSchema),
    defaultValues: {
      method: 'email',
      email: '',
      document_type: 'cedula_ciudadania',
      document_number: '',
      user_id: '',
    },
  })
  const method = useWatch({ control, name: 'method' })

  const mutation = useMutation({
    mutationFn: (payload: CreateInvitationPayload) => createFundInvitation(fundId, payload),
    onSuccess: async () => {
      setServerError('')
      setSuccess('Invitación enviada correctamente.')
      reset()
      await queryClient.invalidateQueries({ queryKey: ['fund-invitations', fundId] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
    onError: (err: unknown) => {
      setSuccess('')
      setServerError(parseApiError(err))
    },
  })

  function onSubmit(values: AddMemberValues) {
    setServerError('')
    setSuccess('')
    if (values.method === 'email') mutation.mutate({ email: values.email })
    if (values.method === 'document') {
      mutation.mutate({
        document_type: values.document_type,
        document_number: values.document_number,
      })
    }
    if (values.method === 'user_id') mutation.mutate({ user_id: values.user_id })
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <SelectField label="Buscar por" {...register('method')}>
        <option value="email">Correo</option>
        <option value="document">Documento</option>
        <option value="user_id">ID de usuario</option>
      </SelectField>

      {method === 'email' && (
        <Input label="Correo del usuario" placeholder="persona@ejemplo.com" error={errors.email?.message} {...register('email')} />
      )}

      {method === 'document' && (
        <div className="grid gap-3 sm:grid-cols-[1fr_1.2fr]">
          <SelectField label="Tipo" {...register('document_type')}>
            <option value="cedula_ciudadania">Cédula C.</option>
            <option value="cedula_extranjeria">Cédula E.</option>
            <option value="pasaporte">Pasaporte</option>
          </SelectField>
          <Input label="Número" error={errors.document_number?.message} {...register('document_number')} />
        </div>
      )}

      {method === 'user_id' && (
        <Input label="UUID del usuario" error={errors.user_id?.message} {...register('user_id')} />
      )}

      {serverError && <p className="rounded-lg bg-danger/10 px-3 py-2 text-sm text-danger">{serverError}</p>}
      {success && <p className="rounded-lg bg-success/10 px-3 py-2 text-sm text-success">{success}</p>}

      <Button type="submit" loading={mutation.isPending} className="w-full">
        <Mail className="h-4 w-4" />
        Enviar invitación
      </Button>
    </form>
  )
}

function InvitationList({ invitations }: { invitations?: FundInvitation[] | null }) {
  const safeInvitations = Array.isArray(invitations) ? invitations : []

  if (safeInvitations.length === 0) {
    return <p className="text-sm text-slate-400">No hay invitaciones para este fondo.</p>
  }

  return (
    <div className="divide-y divide-navy-600/70">
      {safeInvitations.map((invitation) => (
        <div key={invitation.id} className="flex items-center justify-between gap-4 py-4">
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-white">{invitation.invitee_id}</p>
            <p className="text-xs text-slate-500">Expira {formatDate(invitation.expires_at)}</p>
          </div>
          <InvitationBadge status={invitation.status} />
        </div>
      ))}
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

  return <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${className}`}>{label}</span>
}

function RulesGrid({ fund }: { fund: Fund }) {
  const rules = fund.rules
  const rows = [
    ['Tipo', FUND_TYPE_LABEL[fund.type]],
    ['Estado', FUND_STATUS_LABEL[fund.status]],
    ['Aporte', formatCOP(rules?.contribution_amount ?? '0')],
    ['Frecuencia', frequencyLabel(rules?.frequency ?? 'monthly')],
    ['Periodos', String(rules?.total_periods ?? 0)],
    ['Miembros requeridos', `${rules?.min_members ?? 0} mínimo, ${rules?.max_members ?? 0} máximo`],
    ['Gobernanza', governanceLabel(rules?.governance_type ?? 'admin_only')],
    ['Votación', `${rules?.voting_deadline_hours ?? 0} horas`],
    ['Mora', rules?.penalty_enabled ? `${formatCOP(rules.penalty_amount)} después de ${rules.grace_period_days} días` : 'Sin mora'],
  ]

  return (
    <div className="grid gap-3 md:grid-cols-2">
      {rows.map(([label, value]) => (
        <div key={label} className="rounded-lg bg-navy-950/40 px-4 py-3">
          <p className="text-xs text-slate-500">{label}</p>
          <p className="mt-1 text-sm font-medium text-white">{value}</p>
        </div>
      ))}
    </div>
  )
}

function MembersList({ members, creatorId }: { members: FundMember[]; creatorId: string }) {
  if (members.length === 0) {
    return <p className="text-sm text-slate-400">No hay miembros activos.</p>
  }

  return (
    <div className="divide-y divide-navy-600/70">
      {members.map((member) => (
        <div key={member.id} className="flex items-center justify-between gap-4 py-4">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-cyan/15 text-sm font-bold text-brand-cyan">
              {member.role === 'admin' ? 'A' : 'M'}
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-white">
                {member.user_id === creatorId ? 'Creador del fondo' : 'Miembro'}
              </p>
              <p className="truncate text-xs text-slate-500">{member.user_id}</p>
            </div>
          </div>
          <div className="text-right">
            <p className="text-xs font-medium text-slate-300">{member.role === 'admin' ? 'Admin' : 'Miembro'}</p>
            <p className="mt-1 text-xs text-slate-500">{formatDate(member.joined_at)}</p>
          </div>
        </div>
      ))}
    </div>
  )
}

function Metric({ icon: Icon, label, value }: { icon: ComponentType<{ className?: string }>; label: string; value: string }) {
  return (
    <article className="rounded-lg border border-navy-600 bg-navy-800 p-5">
      <Icon className="h-5 w-5 text-brand-cyan" />
      <p className="mt-4 text-sm text-slate-400">{label}</p>
      <p className="mt-1 truncate text-xl font-bold text-white">{value}</p>
    </article>
  )
}

function SectionTitle({ icon: Icon, title }: { icon: ComponentType<{ className?: string }>; title: string }) {
  return (
    <div className="mb-4 flex items-center gap-2">
      <Icon className="h-4 w-4 text-brand-cyan" />
      <h2 className="font-semibold text-white">{title}</h2>
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

const PRODUCT_TITLE: Record<string, string> = {
  circulo:      'Círculo de ahorro',
  vaca:         'Vaca',
  fondo_ahorro: 'Fondo de ahorro',
}

const PROPOSAL_TYPE_LABEL: Record<ProposalType, string> = {
  activate_fund:     'Activar fondo',
  change_rules:      'Cambiar reglas',
  waive_payment:     'Condonar pago',
  remove_member:     'Retirar miembro',
  cancel_fund:       'Cancelar fondo',
  change_governance: 'Cambiar gobernanza',
  distribute_vaca:   'Distribuir vaca',
}

const PROPOSAL_STATUS_CFG: Record<ProposalStatus, { label: string; cls: string }> = {
  open:     { label: 'Abierta',   cls: 'bg-brand-cyan/15 text-brand-cyan' },
  approved: { label: 'Aprobada',  cls: 'bg-success/15 text-success' },
  rejected: { label: 'Rechazada', cls: 'bg-danger/15 text-danger' },
  expired:  { label: 'Expirada',  cls: 'bg-slate-500/15 text-slate-400' },
}

function GovernanceSection({
  fundId,
  isAdmin,
  fundType,
  members,
}: {
  fundId: string
  isAdmin: boolean
  fundType: string
  members: FundMember[]
}) {
  const [createOpen, setCreateOpen] = useState(false)
  const { data: proposals = [], isLoading } = useQuery({
    queryKey: ['proposals', fundId],
    queryFn: () => listProposals(fundId),
  })

  return (
    <>
      {isAdmin && (
        <div className="mb-4 flex justify-end">
          <Button size="sm" variant="outline" onClick={() => setCreateOpen(true)}>
            <FileText className="h-3.5 w-3.5" />
            Nueva propuesta
          </Button>
        </div>
      )}

      {isLoading ? (
        <div className="flex min-h-24 items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
        </div>
      ) : proposals.length === 0 ? (
        <p className="text-sm text-slate-400">No hay propuestas para este fondo.</p>
      ) : (
        <div className="divide-y divide-navy-600/70">
          {proposals.map(p => (
            <ProposalItem key={p.id} proposal={p} fundId={fundId} />
          ))}
        </div>
      )}

      {createOpen && (
        <CreateProposalModal
          fundId={fundId}
          fundType={fundType}
          members={members}
          onClose={() => setCreateOpen(false)}
        />
      )}
    </>
  )
}

function ProposalItem({ proposal, fundId }: { proposal: Proposal; fundId: string }) {
  const qc = useQueryClient()

  const { data: detail } = useQuery({
    queryKey: ['proposal', fundId, proposal.id],
    queryFn: () => getProposal(fundId, proposal.id),
    enabled: proposal.status === 'open',
  })

  const isOpen   = proposal.status === 'open'
  const myVote   = detail?.my_vote
  const progressPct = proposal.quorum_needed > 0
    ? Math.min(100, Math.round((proposal.votes_for / proposal.quorum_needed) * 100))
    : 0

  const voteMut = useMutation({
    mutationFn: (choice: VoteChoice) => castVote(fundId, proposal.id, choice),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['proposals', fundId] })
      qc.invalidateQueries({ queryKey: ['proposal', fundId, proposal.id] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })

  const { label, cls } = PROPOSAL_STATUS_CFG[proposal.status]

  return (
    <div className="py-4">
      <div className="flex items-start gap-4">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium text-white">{PROPOSAL_TYPE_LABEL[proposal.type]}</span>
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>{label}</span>
          </div>
          <p className="mt-1 text-xs text-slate-500">
            Vence {formatDate(proposal.deadline_at)} · {proposal.votes_for} a favor / {proposal.votes_against} en contra · quórum {proposal.quorum_needed}
          </p>
          <div className="mt-2 h-1.5 w-full max-w-xs overflow-hidden rounded-full bg-navy-950/60">
            <div className="h-full rounded-full bg-brand-cyan transition-all" style={{ width: `${progressPct}%` }} />
          </div>
        </div>

        {isOpen && (
          myVote ? (
            <span className={`flex-shrink-0 rounded-full px-2.5 py-1 text-xs font-medium ${myVote.choice === 'yes' ? 'bg-success/15 text-success' : 'bg-danger/15 text-danger'}`}>
              Voté: {myVote.choice === 'yes' ? 'Sí' : 'No'}
            </span>
          ) : (
            <div className="flex flex-shrink-0 gap-2">
              <Button
                size="sm"
                loading={voteMut.isPending && voteMut.variables === 'yes'}
                disabled={voteMut.isPending}
                onClick={() => voteMut.mutate('yes')}
              >
                <ThumbsUp className="h-3.5 w-3.5" /> Sí
              </Button>
              <Button
                size="sm"
                variant="outline"
                loading={voteMut.isPending && voteMut.variables === 'no'}
                disabled={voteMut.isPending}
                onClick={() => voteMut.mutate('no')}
                className="hover:border-danger/50 hover:text-danger"
              >
                <ThumbsDown className="h-3.5 w-3.5" /> No
              </Button>
            </div>
          )
        )}
      </div>
    </div>
  )
}

const editRulesSchema = z.object({
  contribution_amount: z.string().regex(/^\d+(\.\d{1,2})?$/, 'Monto inválido'),
  frequency: z.enum(['weekly', 'biweekly', 'monthly']),
  total_periods: z.number().int().min(1, 'Mínimo 1 período'),
  penalty_enabled: z.boolean(),
  penalty_type: z.enum(['fixed', 'percentage']),
  penalty_amount: z.string().optional(),
  grace_period_days: z.number().int().min(0),
})
type EditRulesValues = z.infer<typeof editRulesSchema>

function EditRulesModal({
  fund,
  onClose,
  onSaved,
}: {
  fund: Fund
  onClose: () => void
  onSaved: (updated: Fund) => void
}) {
  const [serverError, setServerError] = useState('')
  const r = fund.rules

  const { register, handleSubmit, watch, formState: { errors } } = useForm<EditRulesValues>({
    resolver: zodResolver(editRulesSchema),
    defaultValues: {
      contribution_amount: r?.contribution_amount ?? '0',
      frequency: r?.frequency ?? 'monthly',
      total_periods: r?.total_periods ?? 12,
      penalty_enabled: r?.penalty_enabled ?? false,
      penalty_type: r?.penalty_type ?? 'fixed',
      penalty_amount: r?.penalty_amount ?? '0',
      grace_period_days: r?.grace_period_days ?? 3,
    },
  })
  const penaltyEnabled = watch('penalty_enabled')

  const mutation = useMutation({
    mutationFn: (values: EditRulesValues) =>
      updateFundRules(fund.id, {
        // Pass through immutable fields so binding:"required" passes
        name: fund.name,
        type: fund.type,
        start_date: r?.start_date ?? new Date().toISOString().slice(0, 10),
        min_members: r?.min_members ?? 1,
        max_members: r?.max_members ?? 10,
        governance_type: r?.governance_type ?? 'admin_only',
        voting_deadline_hours: r?.voting_deadline_hours ?? 48,
        // Editable fields
        contribution_amount: values.contribution_amount,
        frequency: values.frequency,
        total_periods: values.total_periods,
        penalty_enabled: values.penalty_enabled,
        penalty_type: values.penalty_type,
        penalty_amount: values.penalty_amount ?? '0',
        grace_period_days: values.grace_period_days,
      }),
    onSuccess: onSaved,
    onError: (err: unknown) => setServerError(parseApiError(err)),
  })

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/60 p-4 backdrop-blur-sm">
      <div className="my-8 w-full max-w-lg rounded-xl border border-navy-600 bg-navy-800 shadow-2xl">
        <div className="flex items-center justify-between border-b border-navy-600 px-5 py-4">
          <div>
            <h2 className="font-semibold text-white">Editar reglas del fondo</h2>
            <p className="text-xs text-slate-500 mt-0.5">Solo disponible mientras el fondo esté en borrador</p>
          </div>
          <button type="button" onClick={onClose} className="rounded-lg p-2 text-slate-400 hover:bg-navy-700 hover:text-white">
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit((v) => mutation.mutate(v))} className="space-y-5 p-5">
          <div className="grid gap-4 sm:grid-cols-3">
            <Input
              label="Cuota"
              inputMode="decimal"
              error={errors.contribution_amount?.message}
              {...register('contribution_amount')}
            />
            <SelectField label="Frecuencia" error={errors.frequency?.message} {...register('frequency')}>
              <option value="weekly">Semanal</option>
              <option value="biweekly">Quincenal</option>
              <option value="monthly">Mensual</option>
            </SelectField>
            <Input
              label="Períodos"
              type="number"
              min={1}
              error={errors.total_periods?.message}
              {...register('total_periods', { valueAsNumber: true })}
            />
          </div>

          <label className="flex items-center gap-3 rounded-lg border border-navy-600 bg-navy-950/40 px-4 py-3 text-sm text-slate-300 cursor-pointer">
            <input type="checkbox" className="h-4 w-4 accent-brand-cta" {...register('penalty_enabled')} />
            Aplicar mora por pagos tarde
          </label>

          {penaltyEnabled && (
            <div className="grid gap-4 sm:grid-cols-3">
              <SelectField label="Tipo mora" {...register('penalty_type')}>
                <option value="fixed">Monto fijo</option>
                <option value="percentage">Porcentaje</option>
              </SelectField>
              <Input label="Valor mora" inputMode="decimal" {...register('penalty_amount')} />
              <Input
                label="Días de gracia"
                type="number"
                min={0}
                error={errors.grace_period_days?.message}
                {...register('grace_period_days', { valueAsNumber: true })}
              />
            </div>
          )}

          {serverError && (
            <p className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-sm text-danger">
              {serverError}
            </p>
          )}

          <div className="flex justify-end gap-3 border-t border-navy-600 pt-5">
            <Button type="button" variant="outline" onClick={onClose}>Cancelar</Button>
            <Button type="submit" loading={mutation.isPending}>
              <Pencil className="h-4 w-4" />
              Guardar cambios
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

function CreateProposalModal({
  fundId,
  fundType,
  members,
  onClose,
}: {
  fundId: string
  fundType: string
  members: FundMember[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [type, setType] = useState<ProposalType>('cancel_fund')
  const [governance, setGovernance] = useState<string>('majority')
  const [selectedPaymentId, setSelectedPaymentId] = useState<string>('')
  const [selectedMemberId, setSelectedMemberId] = useState<string>('')
  const [serverError, setServerError] = useState('')

  // Payments for waive_payment picker — only fetched when that type is selected
  const { data: allPayments = [], isLoading: paymentsLoading } = useQuery({
    queryKey: ['payments-all', fundId],
    queryFn: () => getAllPayments(fundId),
    enabled: type === 'waive_payment',
  })
  const waivablePayments = allPayments.filter(
    (p: Payment) => p.status === 'pending' || p.status === 'missed' || p.status === 'partial',
  )

  // Non-admin members for remove_member picker
  const removableMembers = members.filter(m => m.role !== 'admin')

  const availableTypes: { value: ProposalType; label: string }[] = [
    { value: 'cancel_fund',       label: 'Cancelar fondo' },
    { value: 'change_governance', label: 'Cambiar gobernanza' },
    { value: 'change_rules',      label: 'Cambiar reglas' },
    { value: 'waive_payment',     label: 'Condonar pago' },
    { value: 'remove_member',     label: 'Retirar miembro' },
    ...(fundType === 'vaca' ? [{ value: 'distribute_vaca' as ProposalType, label: 'Distribuir vaca' }] : []),
  ]

  const mutation = useMutation({
    mutationFn: () => {
      const payload: Record<string, unknown> = {}
      if (type === 'change_governance') payload.governance_type = governance
      if (type === 'waive_payment')     payload.payment_id = selectedPaymentId
      if (type === 'remove_member')     payload.member_id = selectedMemberId
      return createProposal(fundId, type, payload)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['proposals', fundId] })
      qc.invalidateQueries({ queryKey: ['dashboard'] })
      onClose()
    },
    onError: (err: unknown) => setServerError(parseApiError(err)),
  })

  const isSubmitDisabled =
    (type === 'waive_payment' && !selectedPaymentId) ||
    (type === 'remove_member' && !selectedMemberId)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm">
      <div className="w-full max-w-md rounded-xl border border-navy-600 bg-navy-800 shadow-2xl">
        <div className="flex items-center justify-between border-b border-navy-600 px-5 py-4">
          <h2 className="font-semibold text-white">Nueva propuesta</h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-slate-400 hover:bg-navy-700 hover:text-white"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4 p-5">
          <SelectField
            label="Tipo de propuesta"
            value={type}
            onChange={e => {
              setType(e.target.value as ProposalType)
              setSelectedPaymentId('')
              setSelectedMemberId('')
              setServerError('')
            }}
          >
            {availableTypes.map(({ value, label }) => (
              <option key={value} value={value}>{label}</option>
            ))}
          </SelectField>

          {type === 'change_governance' && (
            <SelectField
              label="Nueva gobernanza"
              value={governance}
              onChange={e => setGovernance(e.target.value)}
            >
              <option value="admin_only">Solo admin</option>
              <option value="majority">Mayoría</option>
              <option value="unanimous">Unánime</option>
            </SelectField>
          )}

          {type === 'waive_payment' && (
            paymentsLoading ? (
              <div className="flex justify-center py-4">
                <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
              </div>
            ) : waivablePayments.length === 0 ? (
              <p className="rounded-lg bg-slate-500/10 px-4 py-3 text-sm text-slate-400">
                No hay pagos pendientes o vencidos para condonar.
              </p>
            ) : (
              <SelectField
                label="Pago a condonar"
                value={selectedPaymentId}
                onChange={e => setSelectedPaymentId(e.target.value)}
              >
                <option value="">Selecciona un pago…</option>
                {waivablePayments.map((p: Payment) => (
                  <option key={p.id} value={p.id}>
                    Período #{p.period_number} — {formatCOP(p.amount_due)} — {formatDate(p.due_date)}
                  </option>
                ))}
              </SelectField>
            )
          )}

          {type === 'remove_member' && (
            removableMembers.length === 0 ? (
              <p className="rounded-lg bg-slate-500/10 px-4 py-3 text-sm text-slate-400">
                No hay miembros que se puedan retirar.
              </p>
            ) : (
              <SelectField
                label="Miembro a retirar"
                value={selectedMemberId}
                onChange={e => setSelectedMemberId(e.target.value)}
              >
                <option value="">Selecciona un miembro…</option>
                {removableMembers.map(m => (
                  <option key={m.id} value={m.id}>
                    {m.payout_order ? `Turno #${m.payout_order}` : 'Miembro'} — {m.user_id.substring(0, 12)}…
                  </option>
                ))}
              </SelectField>
            )
          )}

          {type === 'change_rules' && (
            <p className="rounded-lg bg-brand-blue/10 px-4 py-3 text-sm text-brand-blue">
              Se abrirá una votación para aprobar el cambio de reglas.
            </p>
          )}

          {serverError && (
            <p className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-sm text-danger">
              {serverError}
            </p>
          )}
        </div>

        <div className="flex justify-end gap-3 border-t border-navy-600 px-5 py-4">
          <Button variant="outline" onClick={onClose}>Cancelar</Button>
          <Button
            loading={mutation.isPending}
            disabled={isSubmitDisabled}
            onClick={() => mutation.mutate()}
          >
            Crear propuesta
          </Button>
        </div>
      </div>
    </div>
  )
}

function frequencyLabel(value: string) {
  return { weekly: 'Semanal', biweekly: 'Quincenal', monthly: 'Mensual' }[value] ?? value
}

function governanceLabel(value: string) {
  return { admin_only: 'Solo admin', majority: 'Mayoría', unanimous: 'Unánime' }[value] ?? value
}

function parseApiError(err: unknown) {
  const error = (err as { response?: { data?: { error?: { code?: string; message?: string } } } })?.response?.data?.error
  if (error?.code === 'NOT_ENOUGH_MEMBERS') return 'El fondo no tiene el mínimo de miembros requeridos.'
  if (error?.code === 'ALREADY_MEMBER') return 'Ese usuario ya pertenece al fondo.'
  if (error?.code === 'INVITATION_ALREADY_PENDING') return 'Ese usuario ya tiene una invitación pendiente.'
  if (error?.code === 'SELF_INVITE') return 'No puedes invitarte a ti mismo.'
  if (error?.code === 'NOT_FOUND') return 'No encontré un usuario con esos datos.'
  if (error?.code === 'VALIDATION_ERROR') return 'Datos inválidos. Revisa la información ingresada.'
  if (error?.message) return error.message
  return 'No se pudo completar la operación.'
}

function InvalidFund() {
  return (
    <section className="rounded-lg border border-danger/30 bg-danger/10 p-6">
      <div className="flex gap-3">
        <AlertTriangle className="mt-0.5 h-5 w-5 text-danger" />
        <div>
          <h1 className="font-semibold text-white">Fondo inválido</h1>
          <p className="mt-1 text-sm text-slate-400">Vuelve al listado y selecciona un fondo.</p>
          <Link to="/app/funds">
            <Button className="mt-4" size="sm" variant="outline">
              <ArrowLeft className="h-4 w-4" />
              Volver
            </Button>
          </Link>
        </div>
      </div>
    </section>
  )
}
