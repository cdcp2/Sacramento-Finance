import client from './client'
import type { Fund, FundInvitation, FundMember } from '@/types'

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

export type CreateFundPayload = {
  name: string
  description?: string
  type: 'circulo' | 'vaca' | 'fondo_ahorro'
  contribution_amount: string
  frequency: 'weekly' | 'biweekly' | 'monthly'
  total_periods: number
  start_date: string
  penalty_enabled: boolean
  penalty_type?: 'fixed' | 'percentage'
  penalty_amount?: string
  grace_period_days?: number
  min_members: number
  max_members: number
  governance_type: 'admin_only' | 'majority' | 'unanimous'
  voting_deadline_hours: number
  payout_order_type?: 'fixed' | 'randomized'
  goal_amount?: string
  goal_description?: string
  distribution_type?: 'goal_reached' | 'unanimous_vote'
  interest_rate?: string
  early_withdrawal_penalty?: string
}

export type AddMemberPayload = {
  email?: string
  document_type?: 'cedula_ciudadania' | 'cedula_extranjeria' | 'pasaporte'
  document_number?: string
  user_id?: string
}

export type CreateInvitationPayload = AddMemberPayload & {
  message?: string
  expires_in_days?: number
}

export async function listFunds(): Promise<Fund[]> {
  const { data } = await client.get<{ data: Fund[] | null }>('/funds')
  return arrayOrEmpty(data.data)
}

export async function getFund(fundId: string): Promise<Fund> {
  const { data } = await client.get<{ data: Fund }>(`/funds/${fundId}`)
  return data.data
}

export async function createFund(payload: CreateFundPayload): Promise<Fund> {
  const { data } = await client.post<{ data: Fund }>('/funds', payload)
  return data.data
}

export type UpdateRulesPayload = Pick<
  CreateFundPayload,
  | 'name' | 'type' | 'contribution_amount' | 'frequency' | 'total_periods' | 'start_date'
  | 'penalty_enabled' | 'penalty_type' | 'penalty_amount' | 'grace_period_days'
  | 'min_members' | 'max_members' | 'governance_type' | 'voting_deadline_hours'
>

export async function updateFundRules(fundId: string, payload: UpdateRulesPayload): Promise<Fund> {
  const { data } = await client.patch<{ data: Fund }>(`/funds/${fundId}`, payload)
  return data.data
}

export async function listFundMembers(fundId: string): Promise<FundMember[]> {
  const { data } = await client.get<{ data: FundMember[] | null }>(`/funds/${fundId}/members`)
  return arrayOrEmpty(data.data)
}

export async function addFundMember(fundId: string, payload: AddMemberPayload): Promise<FundMember> {
  const { data } = await client.post<{ data: FundMember }>(`/funds/${fundId}/members`, payload)
  return data.data
}

export async function activateFund(fundId: string): Promise<Fund> {
  const { data } = await client.post<{ data: Fund }>(`/funds/${fundId}/activate`)
  return data.data
}

export async function createFundInvitation(fundId: string, payload: CreateInvitationPayload): Promise<FundInvitation> {
  const { data } = await client.post<{ data: FundInvitation }>(`/funds/${fundId}/invitations`, payload)
  return data.data
}

export async function listFundInvitations(fundId: string): Promise<FundInvitation[]> {
  const { data } = await client.get<{ data: FundInvitation[] | null }>(`/funds/${fundId}/invitations`)
  return arrayOrEmpty(data.data)
}

export async function listMyInvitations(): Promise<FundInvitation[]> {
  const { data } = await client.get<{ data: FundInvitation[] | null }>('/invitations')
  return arrayOrEmpty(data.data)
}

export async function acceptInvitation(invitationId: string): Promise<FundInvitation> {
  const { data } = await client.post<{ data: FundInvitation }>(`/invitations/${invitationId}/accept`)
  return data.data
}

export async function rejectInvitation(invitationId: string): Promise<FundInvitation> {
  const { data } = await client.post<{ data: FundInvitation }>(`/invitations/${invitationId}/reject`)
  return data.data
}
