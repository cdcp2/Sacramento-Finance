import client from './client'
import type { Payment, LedgerEntry } from '@/types'

function arrayOrEmpty<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : []
}

export async function getMyPayments(fundId: string): Promise<Payment[]> {
  const { data } = await client.get<{ data: Payment[] | null }>(`/funds/${fundId}/payments`)
  return arrayOrEmpty(data.data)
}

export async function getAllPayments(fundId: string): Promise<Payment[]> {
  const { data } = await client.get<{ data: Payment[] | null }>(`/funds/${fundId}/payments/all`)
  return arrayOrEmpty(data.data)
}

export async function payPayment(
  fundId: string,
  paymentId: string,
): Promise<{ payment: Payment; entry: LedgerEntry }> {
  const { data } = await client.post<{ data: { payment: Payment; entry: LedgerEntry } }>(
    `/funds/${fundId}/payments/${paymentId}/pay`,
  )
  return data.data
}

export async function waivePayment(fundId: string, paymentId: string): Promise<Payment> {
  const { data } = await client.post<{ data: Payment }>(
    `/funds/${fundId}/payments/${paymentId}/waive`,
  )
  return data.data
}

export async function getLedger(
  fundId: string,
): Promise<{ balance: string; entries: LedgerEntry[] }> {
  const { data } = await client.get<{ data: { balance: string; entries: LedgerEntry[] | null } }>(
    `/funds/${fundId}/ledger`,
  )
  return { balance: data.data.balance, entries: arrayOrEmpty(data.data.entries) }
}
