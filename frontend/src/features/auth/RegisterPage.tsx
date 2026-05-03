import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Link, useNavigate } from 'react-router-dom'
import { register as registerApi, login, getMe } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/ui/Button'
import Input from '@/components/ui/Input'
import SacramentoLogo from '@/components/brand/SacramentoLogo'

// Backend document type values (must match oneof binding in Go handler)
const DOC_TYPES = [
  { value: 'cedula_ciudadania',  label: 'Cédula de Ciudadanía' },
  { value: 'cedula_extranjeria', label: 'Cédula de Extranjería' },
  { value: 'pasaporte',          label: 'Pasaporte' },
] as const

type DocTypeValue = typeof DOC_TYPES[number]['value']

// Human-readable messages for backend error codes
const API_ERRORS: Record<string, string> = {
  DUPLICATE_EMAIL:          'Este correo ya está registrado. ¿Ya tienes cuenta?',
  DUPLICATE_CEDULA:         'Este número de documento ya está registrado.',
  INVALID_DOCUMENT_NUMBER:  'El número de documento no es válido para el tipo seleccionado.',
  VALIDATION_ERROR:         'Datos inválidos. Revisa el formulario.',
  INTERNAL_ERROR:           'Error del servidor. Intenta en unos minutos.',
}

function parseApiError(err: unknown): string {
  const code = (err as { response?: { data?: { error?: { code?: string; message?: string } } } })
    ?.response?.data?.error
  if (code?.code && API_ERRORS[code.code]) return API_ERRORS[code.code]
  if (code?.message) return code.message
  return 'No se pudo crear la cuenta. Intenta de nuevo.'
}

const schema = z
  .object({
    full_name: z.string().min(3, 'Mínimo 3 caracteres'),
    document_type: z.enum(
      ['cedula_ciudadania', 'cedula_extranjeria', 'pasaporte'],
      { message: 'Selecciona un tipo' },
    ),
    document_number: z.string().min(3, 'Número inválido'),
    email: z.string().email('Correo inválido'),
    phone: z
      .string()
      .min(7, 'Mínimo 7 dígitos')
      .max(15, 'Máximo 15 dígitos')
      .regex(/^\d+$/, 'Solo dígitos, sin espacios ni guiones'),
    password: z.string().min(8, 'Mínimo 8 caracteres'),
    confirm_password: z.string(),
  })
  .refine(d => d.password === d.confirm_password, {
    message: 'Las contraseñas no coinciden',
    path: ['confirm_password'],
  })

type FormValues = z.infer<typeof schema>

export default function RegisterPage() {
  const [serverError, setServerError] = useState('')
  const navigate = useNavigate()
  const setUser = useAuthStore(s => s.setUser)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { document_type: 'cedula_ciudadania' },
  })

  async function onSubmit(data: FormValues) {
    setServerError('')
    try {
      await registerApi({
        full_name:       data.full_name,
        document_type:   data.document_type,
        document_number: data.document_number,
        email:           data.email,
        phone:           data.phone,
        password:        data.password,
      })
      const tokens = await login({ email: data.email, password: data.password })
      localStorage.setItem('access_token', tokens.access_token)
      localStorage.setItem('refresh_token', tokens.refresh_token)
      const me = await getMe()
      setUser(me)
      navigate('/app/dashboard')
    } catch (err) {
      setServerError(parseApiError(err))
    }
  }

  return (
    <div className="min-h-screen flex bg-navy-900">
      {/* ── Left panel ── */}
      <div className="hidden lg:flex lg:w-[42%] relative overflow-hidden bg-gradient-to-br from-navy-800 via-navy-900 to-[#0a1520] items-center justify-center p-12">
        <div className="absolute top-1/4 left-1/4 w-64 h-64 rounded-full bg-brand-blue/10 blur-3xl" />
        <div className="absolute bottom-1/3 right-1/4 w-48 h-48 rounded-full bg-brand-cyan/10 blur-3xl" />
        <div className="absolute top-1/2 right-1/3 w-32 h-32 rounded-full bg-brand-green/10 blur-2xl" />

        <div className="relative z-10 max-w-xs text-center space-y-8">
          <div className="flex justify-center">
            <SacramentoLogo size="lg" />
          </div>
          <div className="space-y-3">
            <h2 className="text-2xl font-bold text-white">Empieza hoy</h2>
            <p className="text-slate-400 text-sm leading-relaxed">
              Crea tu cuenta y únete a grupos de ahorro con personas de confianza.
            </p>
          </div>
          <div className="space-y-2 text-left text-sm text-slate-400">
            {[
              'Registro gratuito, sin cuotas de entrada',
              'Fondos, vacas y círculos en un solo lugar',
              'Historial y reportes en tiempo real',
            ].map(t => (
              <p key={t} className="flex items-start gap-2">
                <span className="text-brand-cyan mt-0.5">✓</span> {t}
              </p>
            ))}
          </div>
        </div>
      </div>

      {/* ── Right form panel ── */}
      <div className="flex-1 flex items-start lg:items-center justify-center p-6 overflow-y-auto">
        <div className="w-full max-w-md py-8 space-y-6">
          <div className="lg:hidden flex justify-center">
            <SacramentoLogo size="sm" />
          </div>

          <div className="space-y-1">
            <h1 className="text-2xl font-bold text-white">Crear cuenta</h1>
            <p className="text-slate-400 text-sm">Completa tu información para comenzar</p>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <Input
              label="Nombre completo"
              placeholder="María García López"
              error={errors.full_name?.message}
              {...register('full_name')}
            />

            {/* Document type (full-width select) + number */}
            <div className="space-y-1">
              <label className="text-sm font-medium text-slate-300">Tipo de documento</label>
              <select
                {...register('document_type')}
                className="w-full rounded-lg bg-navy-950 border border-navy-600 px-3.5 py-2.5 text-sm text-white focus:outline-none focus:ring-2 focus:ring-brand-cyan focus:border-transparent"
              >
                {DOC_TYPES.map(d => (
                  <option key={d.value} value={d.value as DocTypeValue}>{d.label}</option>
                ))}
              </select>
              {errors.document_type && (
                <p className="text-xs text-danger">{errors.document_type.message}</p>
              )}
            </div>

            <Input
              label="Número de documento"
              placeholder="1234567890"
              error={errors.document_number?.message}
              {...register('document_number')}
            />

            <Input
              label="Correo electrónico"
              type="email"
              placeholder="nombre@ejemplo.com"
              error={errors.email?.message}
              {...register('email')}
            />

            <Input
              label="Teléfono (solo dígitos)"
              type="tel"
              placeholder="3001234567"
              hint="Sin espacios ni guiones — ej. 3001234567"
              error={errors.phone?.message}
              {...register('phone')}
            />

            <div className="grid grid-cols-2 gap-3">
              <Input
                label="Contraseña"
                type="password"
                placeholder="••••••••"
                hint="Mínimo 8 caracteres"
                error={errors.password?.message}
                {...register('password')}
              />
              <Input
                label="Confirmar"
                type="password"
                placeholder="••••••••"
                error={errors.confirm_password?.message}
                {...register('confirm_password')}
              />
            </div>

            {serverError && (
              <p className="text-sm text-danger bg-danger/10 border border-danger/20 rounded-lg px-3 py-2">
                {serverError}
              </p>
            )}

            <Button type="submit" size="lg" loading={isSubmitting} className="w-full">
              Crear cuenta
            </Button>
          </form>

          <p className="text-center text-sm text-slate-400">
            ¿Ya tienes cuenta?{' '}
            <Link to="/login" className="text-brand-cyan hover:underline font-medium">
              Inicia sesión
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
