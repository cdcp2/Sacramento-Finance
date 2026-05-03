// ── Auth ─────────────────────────────────────────────────────────────────────

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface User {
  id: string
  document_type: 'cedula_ciudadania' | 'cedula_extranjeria' | 'pasaporte'
  document_number: string
  email: string
  phone: string
  full_name: string
  is_verified: boolean
  created_at: string
}

// ── Funds ────────────────────────────────────────────────────────────────────

export type FundType = 'circulo' | 'vaca' | 'fondo_ahorro'
export type FundStatus = 'draft' | 'active' | 'paused' | 'completed' | 'cancelled'
export type GovernanceType = 'admin_only' | 'majority' | 'unanimous'
export type PeriodFrequency = 'weekly' | 'biweekly' | 'monthly'

export interface FundRules {
  contribution_amount: string
  frequency: PeriodFrequency
  total_periods: number
  start_date: string
  penalty_enabled: boolean
  penalty_type: 'fixed' | 'percentage'
  penalty_amount: string
  grace_period_days: number
  min_members: number
  max_members: number
  governance_type: GovernanceType
  voting_deadline_hours: number
}

export interface Fund {
  id: string
  name: string
  description: string
  type: FundType
  status: FundStatus
  creator_id: string
  currency: string
  rules: FundRules
  created_at: string
  updated_at: string
}

export type MemberRole = 'admin' | 'member'
export type MemberStatus = 'active' | 'suspended' | 'left'

export interface FundMember {
  id: string
  fund_id: string
  user_id: string
  role: MemberRole
  status: MemberStatus
  payout_order?: number
  joined_at: string
}

// ── Product configs ───────────────────────────────────────────────────────────

export interface CirculoConfig {
  fund_id: string
  payout_order_type: 'fixed' | 'randomized'
  current_round: number
  rounds_completed: number
}

export interface VacaConfig {
  fund_id: string
  goal_amount: string
  goal_description: string
  distribution_type: 'goal_reached' | 'unanimous_vote'
}

export interface FondoConfig {
  fund_id: string
  interest_rate: string
  early_withdrawal_penalty: string
}

// ── Payments & Ledger ─────────────────────────────────────────────────────────

export type PaymentStatus = 'pending' | 'partial' | 'paid' | 'missed' | 'waived'

export interface Payment {
  id: string
  fund_id: string
  member_id: string
  period_number: number
  due_date: string
  amount_due: string
  amount_paid: string
  status: PaymentStatus
  paid_at?: string
  penalty_applied: boolean
  penalty_amount: string
  created_at: string
  updated_at: string
}

export type LedgerDirection = 'credit' | 'debit'
export type LedgerEntryType = 'deposit' | 'withdrawal' | 'penalty' | 'payout' | 'interest' | 'refund'

export interface LedgerEntry {
  id: string
  fund_id: string
  user_id?: string
  type: LedgerEntryType
  direction: LedgerDirection
  amount: string
  balance_after: string
  description: string
  created_at: string
}

// ── Governance ────────────────────────────────────────────────────────────────

export type ProposalType =
  | 'activate_fund'
  | 'change_rules'
  | 'waive_payment'
  | 'remove_member'
  | 'cancel_fund'
  | 'change_governance'
  | 'distribute_vaca'

export type ProposalStatus = 'open' | 'approved' | 'rejected' | 'expired'
export type VoteChoice = 'yes' | 'no'

export interface Proposal {
  id: string
  fund_id: string
  proposer_id: string
  type: ProposalType
  status: ProposalStatus
  payload: Record<string, unknown>
  votes_for: number
  votes_against: number
  quorum_needed: number
  total_members: number
  deadline_at: string
  resolved_at?: string
  created_at: string
}

export interface Vote {
  id: string
  proposal_id: string
  member_id: string
  choice: VoteChoice
  created_at: string
}

// ── Notifications ─────────────────────────────────────────────────────────────

export type NotificationType =
  | 'payment_confirmed' | 'payment_waived' | 'payment_overdue'
  | 'payout_received' | 'round_closed'
  | 'withdrawal' | 'interest_accrued' | 'goal_reached'
  | 'member_joined' | 'member_left'
  | 'proposal_created' | 'proposal_resolved'

export interface Notification {
  id: string
  user_id: string
  fund_id?: string
  type: NotificationType
  title: string
  body: string
  is_read: boolean
  created_at: string
}

// ── API response wrappers ─────────────────────────────────────────────────────

export interface ApiResponse<T> {
  data: T
}

export interface ApiError {
  code: string
  message: string
}

// ── Dashboard ─────────────────────────────────────────────────────────────────

export interface DashboardData {
  funds: Fund[]
  pending_payments: Payment[]
  open_proposals: Proposal[]
  unread_notifications: number
}
