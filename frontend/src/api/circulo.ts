import client from './client'
import type { CirculoDetail, FundMember, Payout } from '@/types'

export async function getCirculoConfig(fundId: string): Promise<CirculoDetail> {
  const { data } = await client.get<{ data: CirculoDetail }>(`/funds/${fundId}/circulo`)
  return data.data
}

export async function assignPayoutOrder(
  fundId: string,
  options: { randomize: boolean; assignments?: { member_id: string; order: number }[] },
): Promise<FundMember[]> {
  const { data } = await client.post<{ data: FundMember[] }>(
    `/funds/${fundId}/circulo/payout-order`,
    options,
  )
  return data.data
}

export async function listPayouts(fundId: string): Promise<Payout[]> {
  const { data } = await client.get<{ data: Payout[] | null }>(
    `/funds/${fundId}/circulo/payouts`,
  )
  return Array.isArray(data.data) ? data.data : []
}

export async function closeRound(
  fundId: string,
): Promise<{ payout: Payout; config: { current_round: number }; completed: boolean }> {
  const { data } = await client.post<{
    data: { payout: Payout; config: { current_round: number }; completed: boolean }
  }>(`/funds/${fundId}/circulo/close-round`)
  return data.data
}
