import client from './client'
import type { Notification } from '@/types'

export interface NotificationList {
  notifications: Notification[]
  unread_count: number
}

export async function getNotifications(limit = 20, offset = 0): Promise<NotificationList> {
  const { data } = await client.get<{ data: NotificationList }>('/notifications', {
    params: { limit, offset },
  })
  return {
    notifications: Array.isArray(data.data.notifications) ? data.data.notifications : [],
    unread_count: data.data.unread_count ?? 0,
  }
}

export async function markRead(notifId: string): Promise<void> {
  await client.patch(`/notifications/${notifId}/read`)
}

export async function markAllRead(): Promise<void> {
  await client.post('/notifications/read-all')
}
