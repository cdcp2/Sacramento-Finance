import client from './client'
import type { User } from '@/types'

export async function updateMe(data: { full_name?: string; phone?: string }): Promise<User> {
  const { data: res } = await client.patch<{ data: User }>('/users/me', data)
  return res.data
}
