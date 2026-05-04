import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useAuthStore } from '@/stores/auth'
import ErrorBoundary from '@/components/ErrorBoundary'

// Layouts
import AppLayout from '@/components/layout/AppLayout'

// Auth pages
import LoginPage from '@/features/auth/LoginPage'
import RegisterPage from '@/features/auth/RegisterPage'

// App pages
import DashboardPage from '@/features/dashboard/DashboardPage'
import FundsPage from '@/features/funds/FundsPage'
import FundDetailPage from '@/features/funds/FundDetailPage'
import PaymentsPage from '@/features/payments/PaymentsPage'
import NotificationsPage from '@/features/notifications/NotificationsPage'
import ProfilePage from '@/features/profile/ProfilePage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1 },
  },
})

function RequireAuth({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const logout = useAuthStore((s) => s.logout)

  useEffect(() => {
    const hasAccessToken = Boolean(localStorage.getItem('access_token'))
    if (isAuthenticated && !hasAccessToken) {
      logout()
    }
  }, [isAuthenticated, logout])

  if (!isAuthenticated || !localStorage.getItem('access_token')) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />

            <Route
              path="/app"
              element={
                <RequireAuth>
                  <AppLayout />
                </RequireAuth>
              }
            >
              <Route index element={<Navigate to="/app/dashboard" replace />} />
              <Route path="dashboard" element={<DashboardPage />} />
              <Route path="funds" element={<FundsPage />} />
              <Route path="funds/:fundId" element={<FundDetailPage />} />
              <Route path="funds/:fundId/payments" element={<PaymentsPage />} />
              <Route path="notifications" element={<NotificationsPage />} />
              <Route path="profile" element={<ProfilePage />} />
            </Route>

            <Route path="/" element={<Navigate to="/app/dashboard" replace />} />
            <Route path="*" element={<Navigate to="/app/dashboard" replace />} />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}
