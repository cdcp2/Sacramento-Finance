import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Formatea un decimal string como COP: "1200000" → "$1.200.000" */
export function formatCOP(value: string | number): string {
  const n = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(n)) return '$0'
  return new Intl.NumberFormat('es-CO', {
    style: 'currency',
    currency: 'COP',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(n)
}

/** Formatea fecha ISO a "12 may 2025" */
export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat('es-CO', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(new Date(iso))
}

/** Formatea fecha relativa: "hace 2 días" */
export function formatRelative(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'ahora'
  if (mins < 60) return `hace ${mins} min`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `hace ${hours}h`
  const days = Math.floor(hours / 24)
  if (days < 7) return `hace ${days} días`
  return formatDate(iso)
}

export const FUND_TYPE_LABEL: Record<string, string> = {
  circulo:      'Círculo',
  vaca:         'Vaca',
  fondo_ahorro: 'Fondo de Ahorro',
}

export const FUND_STATUS_LABEL: Record<string, string> = {
  draft:     'Borrador',
  active:    'Activo',
  paused:    'Pausado',
  completed: 'Completado',
  cancelled: 'Cancelado',
}

export const PAYMENT_STATUS_LABEL: Record<string, string> = {
  pending: 'Pendiente',
  partial: 'Parcial',
  paid:    'Pagado',
  missed:  'En mora',
  waived:  'Exonerado',
}
