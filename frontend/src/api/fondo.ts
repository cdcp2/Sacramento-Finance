import client from './client'
import type { FondoConfig, LedgerEntry, WithdrawPreview } from '@/types'

export async function getFondoConfig(fundId: string): Promise<FondoConfig> {
  const { data } = await client.get<{ data: FondoConfig }>(`/funds/${fundId}/fondo`)
  return data.data
}

export async function getMyFondoBalance(fundId: string): Promise<string> {
  const { data } = await client.get<{ data: { balance: string } }>(
    `/funds/${fundId}/fondo/my-balance`,
  )
  return data.data.balance
}

export async function getWithdrawalPreview(
  fundId: string,
  amount: string,
): Promise<WithdrawPreview> {
  const { data } = await client.get<{ data: WithdrawPreview }>(
    `/funds/${fundId}/fondo/withdrawal-preview`,
    { params: { amount } },
  )
  return data.data
}

export async function accrueInterest(fundId: string): Promise<LedgerEntry> {
  const { data } = await client.post<{ data: LedgerEntry }>(
    `/funds/${fundId}/fondo/accrue-interest`,
  )
  return data.data
}

export async function withdrawFondo(
  fundId: string,
  amount: string,
): Promise<{ withdrawal_entry: LedgerEntry; penalty_entry?: LedgerEntry }> {
  const { data } = await client.post<{
    data: { withdrawal_entry: LedgerEntry; penalty_entry?: LedgerEntry }
  }>(`/funds/${fundId}/fondo/withdraw`, { amount })
  return data.data
}
