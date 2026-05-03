import client from './client'
import type { TokenPair, User } from '@/types'

export interface LoginPayload {
  email?: string
  document_type?: string
  document_number?: string
  password: string
}

export interface RegisterPayload {
  document_type: string
  document_number: string
  email: string
  phone: string
  full_name: string
  password: string
}

export async function login(payload: LoginPayload): Promise<TokenPair> {
  const { data } = await client.post<{ data: TokenPair }>('/auth/login', payload)
  return data.data
}

export async function register(payload: RegisterPayload): Promise<User> {
  const { data } = await client.post<{ data: User }>('/auth/register', payload)
  return data.data
}

export async function getMe(): Promise<User> {
  const { data } = await client.get<{ data: User }>('/users/me')
  return data.data
}
