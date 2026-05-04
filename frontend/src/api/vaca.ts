import client from './client'
import type { VacaProgress } from '@/types'

export async function getVacaProgress(fundId: string): Promise<VacaProgress> {
  const { data } = await client.get<{ data: VacaProgress }>(`/funds/${fundId}/vaca`)
  return data.data
}

export async function distributeVaca(fundId: string): Promise<{ share_per_member: string }> {
  const { data } = await client.post<{ data: { share_per_member: string } }>(
    `/funds/${fundId}/vaca/distribute`,
  )
  return data.data
}
