import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Link, useNavigate } from 'react-router-dom'
import { login, getMe } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import Button from '@/components/ui/Button'
import Input from '@/components/ui/Input'
import SacramentoLogo from '@/components/brand/SacramentoLogo'

const schema = z.object({
  email: z.string().email('Correo inválido'),
  password: z.string().min(6, 'Mínimo 6 caracteres'),
})
type FormValues = z.infer<typeof schema>

export default function LoginPage() {
  const [serverError, setServerError] = useState('')
  const navigate = useNavigate()
  const setUser = useAuthStore(s => s.setUser)

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  })

  async function onSubmit(data: FormValues) {
    setServerError('')
    try {
      const tokens = await login({ email: data.email, password: data.password })
      localStorage.setItem('access_token', tokens.access_token)
      localStorage.setItem('refresh_token', tokens.refresh_token)
      const me = await getMe()
      setUser(me)
      navigate('/app/dashboard')
    } catch {
      setServerError('Credenciales incorrectas. Verifica tus datos.')
    }
  }

  return (
    <div className="min-h-screen flex bg-navy-900">
      {/* ── Left decorative panel ── */}
      <div className="hidden lg:flex lg:w-1/2 relative overflow-hidden bg-gradient-to-br from-navy-800 via-navy-900 to-[#0a1520] items-center justify-center p-12">
        <div className="absolute top-1/4 left-1/4 w-64 h-64 rounded-full bg-brand-blue/10 blur-3xl" />
        <div className="absolute bottom-1/3 right-1/4 w-48 h-48 rounded-full bg-brand-cyan/10 blur-3xl" />
        <div className="absolute top-1/2 right-1/3 w-32 h-32 rounded-full bg-brand-green/10 blur-2xl" />

        <div className="relative z-10 max-w-sm text-center space-y-8">
          <div className="flex justify-center">
            <SacramentoLogo size="lg" />
          </div>

          <div className="space-y-2">
            <h2 className="text-2xl font-bold text-white">Finanzas colaborativas</h2>
            <p className="text-slate-400 text-sm leading-relaxed">
              Administra fondos grupales, vacas, círculos de ahorro y más — todo en un solo lugar.
            </p>
          </div>

          <div className="grid gap-3 text-left">
            {[
              { icon: '🏦', label: 'Fondos de ahorro grupal' },
              { icon: '🐄', label: 'Vacas con distribución automática' },
              { icon: '🔄', label: 'Círculos rotativos' },
              { icon: '📊', label: 'Historial y reportes en tiempo real' },
            ].map(({ icon, label }) => (
              <div key={label} className="flex items-center gap-3 bg-navy-700/50 rounded-lg px-4 py-2.5 border border-navy-600/50">
                <span className="text-lg">{icon}</span>
                <span className="text-sm text-slate-300">{label}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* ── Right form panel ── */}
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-md space-y-8">
          <div className="lg:hidden flex justify-center">
            <SacramentoLogo size="sm" />
          </div>

          <div className="space-y-1">
            <h1 className="text-2xl font-bold text-white">Iniciar sesión</h1>
            <p className="text-slate-400 text-sm">Bienvenido de vuelta</p>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <Input
              label="Correo electrónico"
              type="email"
              placeholder="nombre@ejemplo.com"
              error={errors.email?.message}
              {...register('email')}
            />
            <Input
              label="Contraseña"
              type="password"
              placeholder="••••••••"
              error={errors.password?.message}
              {...register('password')}
            />

            {serverError && (
              <p className="text-sm text-danger bg-danger/10 border border-danger/20 rounded-lg px-3 py-2">
                {serverError}
              </p>
            )}

            <Button type="submit" size="lg" loading={isSubmitting} className="w-full">
              Ingresar
            </Button>
          </form>

          <p className="text-center text-sm text-slate-400">
            ¿No tienes cuenta?{' '}
            <Link to="/register" className="text-brand-cyan hover:underline font-medium">
              Regístrate
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
