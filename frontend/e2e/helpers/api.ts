/**
 * Thin API helpers for e2e test setup.
 * These hit the backend directly (no browser) to seed the state we need
 * before the UI flow starts.
 */
import type { APIRequestContext } from '@playwright/test'

const API = process.env.API_BASE ?? 'http://localhost:8080/api/v1'

export interface TestUser {
  email: string
  password: string
  fullName: string
  accessToken: string
}

/** Registers a fresh user and returns their credentials + access token. */
export async function registerUser(
  request: APIRequestContext,
  label: string,
): Promise<TestUser> {
  const ts = Date.now()
  const rand = Math.floor(10_000_000 + Math.random() * 89_999_999) // 8-digit doc
  const email = `e2e-${label}-${ts}@test.local`
  const password = 'E2eTest1!'
  const fullName = `E2E ${label.charAt(0).toUpperCase() + label.slice(1)} ${ts}`

  const res = await request.post(`${API}/auth/register`, {
    data: {
      email,
      password,
      full_name: fullName,
      document_type: 'cedula_ciudadania',
      document_number: String(rand),
      phone: '3001234567',
    },
  })

  if (!res.ok()) {
    const body = await res.text()
    throw new Error(`registerUser(${label}) failed ${res.status()}: ${body}`)
  }

  const { data } = await res.json()
  return { email, password, fullName, accessToken: data.access_token }
}

/** Returns a fresh access token for an existing user. */
export async function loginUser(
  request: APIRequestContext,
  email: string,
  password: string,
): Promise<string> {
  const res = await request.post(`${API}/auth/login`, {
    data: { email, password },
  })

  if (!res.ok()) {
    const body = await res.text()
    throw new Error(`loginUser(${email}) failed ${res.status()}: ${body}`)
  }

  const { data } = await res.json()
  return data.access_token
}

export interface TestFund {
  id: string
  name: string
}

/** Creates a draft fund as the given user and returns its id + name. */
export async function createFund(
  request: APIRequestContext,
  accessToken: string,
): Promise<TestFund> {
  const name = `E2E Fund ${Date.now()}`
  const today = new Date().toISOString().slice(0, 10)

  const res = await request.post(`${API}/funds`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    data: {
      name,
      type: 'fondo_ahorro',
      contribution_amount: '50000',
      frequency: 'monthly',
      total_periods: 12,
      start_date: today,
      min_members: 1,
      max_members: 10,
      governance_type: 'admin_only',
      voting_deadline_hours: 48,
      penalty_enabled: false,
      penalty_type: 'fixed',
      penalty_amount: '0',
      grace_period_days: 3,
      interest_rate: '0',
      early_withdrawal_penalty: '0',
    },
  })

  if (!res.ok()) {
    const body = await res.text()
    throw new Error(`createFund failed ${res.status()}: ${body}`)
  }

  const { data } = await res.json()
  return { id: data.id, name: data.name }
}
