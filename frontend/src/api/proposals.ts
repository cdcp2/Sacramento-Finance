import client from './client'
import type { Proposal, Vote, VoteChoice, ProposalType } from '@/types'

export interface ProposalDetail {
  proposal: Proposal
  votes: Vote[]
  my_vote: Vote | null
}

export async function listProposals(fundId: string): Promise<Proposal[]> {
  const { data } = await client.get<{ data: Proposal[] | null }>(`/funds/${fundId}/proposals`)
  return Array.isArray(data.data) ? data.data : []
}

export async function getProposal(fundId: string, proposalId: string): Promise<ProposalDetail> {
  const { data } = await client.get<{ data: ProposalDetail }>(
    `/funds/${fundId}/proposals/${proposalId}`,
  )
  return data.data
}

export async function createProposal(
  fundId: string,
  type: ProposalType,
  payload?: Record<string, unknown>,
): Promise<Proposal> {
  const { data } = await client.post<{ data: Proposal }>(`/funds/${fundId}/proposals`, {
    type,
    payload: payload ?? {},
  })
  return data.data
}

export async function castVote(
  fundId: string,
  proposalId: string,
  choice: VoteChoice,
): Promise<{ proposal: Proposal; resolved: boolean }> {
  const { data } = await client.post<{ data: { proposal: Proposal; resolved: boolean } }>(
    `/funds/${fundId}/proposals/${proposalId}/vote`,
    { choice },
  )
  return data.data
}
