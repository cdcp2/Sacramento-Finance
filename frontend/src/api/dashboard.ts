import client from './client'
import type { DashboardData } from '@/types'

export async function getDashboard(): Promise<DashboardData> {
  const { data } = await client.get<{ data: DashboardData }>('/dashboard')
  return data.data
}
