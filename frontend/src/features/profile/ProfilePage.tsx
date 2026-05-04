import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  CreditCard,
  LogOut,
  Mail,
  Pencil,
  Phone,
  ShieldCheck,
  User,
  X,
} from 'lucide-react'
import { updateMe } from '@/api/users'
import Button from '@/components/ui/Button'
import Input from '@/components/ui/Input'
import { useAuthStore } from '@/stores/auth'

const editSchema = z.object({
  full_name: z.string().min(3, 'Mínimo 3 caracteres').max(100),
  phone: z.string().regex(/^\d{7,15}$/, 'Solo dígitos, 7–15 caracteres'),
})
type EditValues = z.infer<typeof editSchema>

const DOC_TYPE_LABEL: Record<string, string> = {
  cedula_ciudadania:  'Cédula de ciudadanía',
  cedula_extranjeria: 'Cédula de extranjería',
  pasaporte:          'Pasaporte',
}

const VERIFICATION_CFG: Record<string, { label: string; cls: string }> = {
  none:     { label: 'Sin verificar',  cls: 'bg-slate-500/15 text-slate-400' },
  pending:  { label: 'En revisión',    cls: 'bg-warning/15 text-warning' },
  approved: { label: 'Verificado',     cls: 'bg-success/15 text-success' },
  rejected: { label: 'Rechazado',      cls: 'bg-danger/15 text-danger' },
}

export default function ProfilePage() {
  const user = useAuthStore((s) => s.user)
  const setUser = useAuthStore((s) => s.setUser)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()
  const [editing, setEditing] = useState(false)
  const [serverError, setServerError] = useState('')

  const { register, handleSubmit, reset, formState: { errors } } = useForm<EditValues>({
    resolver: zodResolver(editSchema),
    defaultValues: { full_name: user?.full_name ?? '', phone: user?.phone ?? '' },
  })

  const mutation = useMutation({
    mutationFn: updateMe,
    onSuccess: (updated) => {
      setUser(updated)
      setEditing(false)
      setServerError('')
    },
    onError: () => setServerError('No se pudo actualizar el perfil. Intenta de nuevo.'),
  })

  function handleLogout() {
    logout()
    navigate('/login', { replace: true })
  }

  function startEdit() {
    reset({ full_name: user?.full_name ?? '', phone: user?.phone ?? '' })
    setServerError('')
    setEditing(true)
  }

  if (!user) return null

  const verif = VERIFICATION_CFG[user.verification_status ?? 'none']

  return (
    <div className="max-w-2xl mx-auto space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Mi cuenta</h1>
          <p className="mt-1 text-sm text-slate-400">Información de tu perfil</p>
        </div>
        <Button variant="ghost" onClick={handleLogout} className="text-slate-400 hover:text-danger">
          <LogOut className="h-4 w-4" />
          Cerrar sesión
        </Button>
      </div>

      {/* Avatar + name */}
      <div className="flex items-center gap-4 rounded-xl border border-navy-600 bg-navy-700/40 p-5">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-brand-cta/20 text-2xl font-bold text-brand-cta">
          {user.full_name.trim().charAt(0).toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-lg font-semibold text-white">{user.full_name}</p>
          <span className={`mt-1 inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${verif.cls}`}>
            <ShieldCheck className="h-3 w-3" />
            {verif.label}
          </span>
        </div>
        {!editing && (
          <Button size="sm" variant="outline" onClick={startEdit}>
            <Pencil className="h-3.5 w-3.5" />
            Editar
          </Button>
        )}
      </div>

      {/* Edit form */}
      {editing && (
        <form
          onSubmit={handleSubmit((v) => mutation.mutate(v))}
          className="rounded-xl border border-brand-cyan/30 bg-navy-700/40 overflow-hidden"
        >
          <div className="flex items-center justify-between px-5 py-3 border-b border-navy-600">
            <h2 className="text-sm font-semibold text-white">Editar perfil</h2>
            <button
              type="button"
              onClick={() => setEditing(false)}
              className="text-slate-400 hover:text-white"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="space-y-4 p-5">
            <Input label="Nombre completo" error={errors.full_name?.message} {...register('full_name')} />
            <Input label="Teléfono" inputMode="numeric" error={errors.phone?.message} {...register('phone')} />
            {serverError && (
              <p className="rounded-lg border border-danger/20 bg-danger/10 px-3 py-2 text-sm text-danger">
                {serverError}
              </p>
            )}
          </div>
          <div className="flex justify-end gap-3 border-t border-navy-600 px-5 py-4">
            <Button type="button" variant="outline" onClick={() => setEditing(false)}>Cancelar</Button>
            <Button type="submit" loading={mutation.isPending}>Guardar cambios</Button>
          </div>
        </form>
      )}

      {/* Info fields */}
      <div className="rounded-xl border border-navy-600 bg-navy-700/40 overflow-hidden">
        <div className="px-5 py-3 border-b border-navy-600">
          <h2 className="text-sm font-semibold text-white">Datos personales</h2>
        </div>
        <div className="divide-y divide-navy-600/50">
          <InfoRow icon={Mail} label="Correo electrónico" value={user.email} />
          <InfoRow icon={Phone} label="Teléfono" value={user.phone} />
          <InfoRow
            icon={CreditCard}
            label="Documento"
            value={`${DOC_TYPE_LABEL[user.document_type] ?? user.document_type} · ${user.document_number}`}
          />
          <InfoRow icon={User} label="ID de usuario" value={user.id} mono />
        </div>
      </div>
    </div>
  )
}

function InfoRow({
  icon: Icon,
  label,
  value,
  mono = false,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center gap-4 px-5 py-4">
      <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-navy-950/60 text-brand-cyan">
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-slate-500">{label}</p>
        <p className={`mt-0.5 text-sm text-white ${mono ? 'font-mono text-xs text-slate-400' : 'font-medium'}`}>
          {value}
        </p>
      </div>
    </div>
  )
}
