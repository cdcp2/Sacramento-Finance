# Sacramento Finance Backend API

Base URL local:

```text
http://localhost:8080/api/v1
```

Protected routes require:

```http
Authorization: Bearer <access_token>
Content-Type: application/json
```

Standard envelope:

```json
{ "data": {} }
```

Standard error:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "details"
  }
}
```

Frontend note: API responses use `snake_case` JSON fields.

## Enums

Document types:

```text
cedula_ciudadania | cedula_extranjeria | pasaporte
```

Fund types:

```text
circulo | vaca | fondo_ahorro
```

Fund status:

```text
draft | active | paused | completed | cancelled
```

Frequencies:

```text
weekly | biweekly | monthly
```

Governance:

```text
admin_only | majority | unanimous
```

Proposal types:

```text
activate_fund | change_rules | waive_payment | remove_member | cancel_fund | change_governance | distribute_vaca
```

Vote choices:

```text
yes | no
```

## Auth

### Register

```http
POST /auth/register
```

```json
{
  "document_type": "cedula_ciudadania",
  "document_number": "123456789",
  "email": "ana@example.com",
  "phone": "3001234567",
  "full_name": "Ana Perez",
  "password": "supersecret"
}
```

Response `201`:

```json
{
  "data": {
    "id": "uuid",
    "document_type": "cedula_ciudadania",
    "document_number": "123456789",
    "email": "ana@example.com",
    "phone": "3001234567",
    "full_name": "Ana Perez",
    "is_verified": false,
    "verification_status": "none",
    "created_at": "2026-05-01T00:00:00Z"
  }
}
```

Validation:

- `cedula_ciudadania`: digits only, no leading zero, 3 to 10 digits.
- `cedula_extranjeria`: alphanumeric, 4 to 15 chars.
- `pasaporte`: alphanumeric, 5 to 20 chars.

### Login

```http
POST /auth/login
```

By email:

```json
{
  "email": "ana@example.com",
  "password": "supersecret"
}
```

By document:

```json
{
  "document_type": "cedula_ciudadania",
  "document_number": "123456789",
  "password": "supersecret"
}
```

Response `200`:

```json
{
  "data": {
    "access_token": "jwt",
    "refresh_token": "jwt",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

## User

### Get Current User

```http
GET /users/me
```

Response `200`: sanitized user, no password hash.

### Update Current User

```http
PATCH /users/me
```

Only profile fields are editable here:

```json
{
  "full_name": "Ana Maria Perez",
  "phone": "3109876543"
}
```

Response `200`: sanitized user.

## Dashboard

### Get Dashboard

```http
GET /dashboard
```

Response `200`:

```json
{
  "data": {
    "summary": {
      "total_funds": 2,
      "active_funds": 1,
      "draft_funds": 1,
      "completed_funds": 0,
      "admin_funds": 1,
      "pending_payments": 3,
      "overdue_payments": 1,
      "open_proposals": 2,
      "unread_notifications": 4,
      "total_pending_amount": "300000"
    },
    "funds": [
      {
        "id": "uuid",
        "name": "Vaca viaje",
        "type": "vaca",
        "status": "active",
        "role": "admin",
        "governance_type": "majority",
        "pending_payments": 2,
        "overdue_payments": 0,
        "open_proposals": 1,
        "next_payment": {
          "id": "uuid",
          "fund_id": "uuid",
          "fund_name": "Vaca viaje",
          "period_number": 2,
          "due_date": "2026-06-01T00:00:00Z",
          "amount_due": "100000",
          "amount_paid": "0",
          "status": "pending",
          "is_overdue": false
        }
      }
    ],
    "upcoming_payments": [],
    "open_proposals": []
  }
}
```

This is the recommended first call after login.

## Funds

### Create Fund

```http
POST /funds
```

Common body:

```json
{
  "name": "Vaca viaje",
  "description": "Ahorro para viaje",
  "type": "vaca",
  "contribution_amount": "100000",
  "frequency": "monthly",
  "total_periods": 4,
  "start_date": "2026-06-01",
  "penalty_enabled": true,
  "penalty_type": "fixed",
  "penalty_amount": "10000",
  "grace_period_days": 3,
  "min_members": 2,
  "max_members": 10,
  "governance_type": "majority",
  "voting_deadline_hours": 48
}
```

Circulo extras:

```json
{
  "payout_order_type": "fixed"
}
```

Vaca extras:

```json
{
  "goal_amount": "1000000",
  "goal_description": "Viaje grupal",
  "distribution_type": "goal_reached"
}
```

Fondo de ahorro extras:

```json
{
  "interest_rate": "1.5",
  "early_withdrawal_penalty": "2.0"
}
```

Response `201`: created fund. Creator is automatically added as admin member.

### List My Funds

```http
GET /funds
```

Response `200`: funds where caller is a member.

### Get Fund

```http
GET /funds/:fund_id
```

Requires caller to be a fund member.

### Update Rules

```http
PATCH /funds/:fund_id
```

Admin only, draft only, `admin_only` governance only.

For governed funds (`majority` or `unanimous`), create a `change_rules` proposal instead.

### Activate Fund

```http
POST /funds/:fund_id/activate
```

Admin only, `admin_only` governance only. Generates the payment schedule after activation.

For governed funds, create an `activate_fund` proposal instead.

## Members

### List Members

```http
GET /funds/:fund_id/members
```

Requires caller to be a fund member.

### Add Member

```http
POST /funds/:fund_id/members
```

Admin only. You can identify the target user by one of these:

```json
{ "user_id": "uuid" }
```

```json
{ "email": "nuevo@example.com" }
```

```json
{
  "document_type": "cedula_ciudadania",
  "document_number": "987654321"
}
```

Response `201`: created fund member.

## Payments And Ledger

### List My Payments

```http
GET /funds/:fund_id/payments
```

Requires caller to be a fund member.

### List All Fund Payments

```http
GET /funds/:fund_id/payments/all
```

Admin only.

### Pay

```http
POST /funds/:fund_id/payments/:payment_id/pay
```

Requires active fund and payment belonging to caller's membership.

Response `200`:

```json
{
  "data": {
    "payment": {},
    "entry": {}
  }
}
```

### Waive Payment

```http
POST /funds/:fund_id/payments/:payment_id/waive
```

Admin only, `admin_only` governance only.

For governed funds, create a `waive_payment` proposal instead.

### Ledger

```http
GET /funds/:fund_id/ledger
```

Requires caller to be a fund member.

Response includes current balance and last 50 ledger entries.

## Governance

Any active member can create a proposal in governed funds.

### Create Proposal

```http
POST /funds/:fund_id/proposals
```

```json
{
  "type": "activate_fund",
  "payload": {}
}
```

Payloads:

```json
{
  "type": "change_rules",
  "payload": {
    "contribution_amount": "150000",
    "frequency": "monthly",
    "total_periods": 6,
    "penalty_enabled": true,
    "penalty_type": "fixed",
    "penalty_amount": "10000",
    "grace_period_days": 3
  }
}
```

```json
{
  "type": "waive_payment",
  "payload": {
    "payment_id": "uuid"
  }
}
```

```json
{
  "type": "remove_member",
  "payload": {
    "member_id": "uuid",
    "reason": "voluntary exit"
  }
}
```

```json
{
  "type": "change_governance",
  "payload": {
    "governance_type": "unanimous",
    "voting_deadline_hours": 72
  }
}
```

```json
{
  "type": "cancel_fund",
  "payload": {
    "reason": "members agreed to cancel"
  }
}
```

```json
{
  "type": "distribute_vaca",
  "payload": {}
}
```

### List Proposals

```http
GET /funds/:fund_id/proposals
```

Requires caller to be a fund member. Open proposals past deadline are lazily marked expired.

### Get Proposal

```http
GET /funds/:fund_id/proposals/:proposal_id
```

Response includes:

```json
{
  "data": {
    "proposal": {},
    "votes": [],
    "my_vote": {}
  }
}
```

### Vote

```http
POST /funds/:fund_id/proposals/:proposal_id/vote
```

```json
{
  "choice": "yes"
}
```

On approval, the backend automatically executes the proposal action.

## Circulo

### Get Config

```http
GET /funds/:fund_id/circulo
```

Requires member and fund type `circulo`.

### Assign Payout Order

```http
POST /funds/:fund_id/circulo/payout-order
```

Admin only.

Random:

```json
{
  "randomize": true
}
```

Manual:

```json
{
  "assignments": [
    { "member_id": "uuid", "order": 1 },
    { "member_id": "uuid", "order": 2 }
  ]
}
```

### List Payouts

```http
GET /funds/:fund_id/circulo/payouts
```

### Close Round

```http
POST /funds/:fund_id/circulo/close-round
```

Admin only. Creates payout and ledger movement.

## Vaca

### Get Progress

```http
GET /funds/:fund_id/vaca
```

Requires member and fund type `vaca`.

### Distribute

```http
POST /funds/:fund_id/vaca/distribute
```

Admin only, `admin_only` governance only.

For governed funds, create a `distribute_vaca` proposal instead.

## Fondo De Ahorro

### Get Config

```http
GET /funds/:fund_id/fondo
```

### Get My Balance

```http
GET /funds/:fund_id/fondo/my-balance
```

### Withdrawal Preview

```http
GET /funds/:fund_id/fondo/withdrawal-preview?amount=100000
```

### Accrue Interest

```http
POST /funds/:fund_id/fondo/accrue-interest
```

Admin only.

### Withdraw

```http
POST /funds/:fund_id/fondo/withdraw
```

```json
{
  "amount": "50000"
}
```

## Notifications

### List

```http
GET /notifications?limit=20&offset=0
```

Response:

```json
{
  "data": {
    "notifications": [],
    "unread_count": 0
  }
}
```

### Mark Read

```http
PATCH /notifications/:notif_id/read
```

### Mark All Read

```http
POST /notifications/read-all
```

## Governance Rules For The Front

When `governance_type` is:

- `admin_only`: admin can use direct admin endpoints.
- `majority`: direct admin decisions are blocked with `USE_PROPOSAL`; use proposal/vote flow.
- `unanimous`: direct admin decisions are blocked with `USE_PROPOSAL`; all active members must vote yes.

Direct endpoints currently blocked for governed funds:

- `POST /funds/:fund_id/activate`
- `PATCH /funds/:fund_id`
- `POST /funds/:fund_id/payments/:payment_id/waive`
- `POST /funds/:fund_id/vaca/distribute`

## Common Error Codes

```text
UNAUTHORIZED
INVALID_TOKEN
VALIDATION_ERROR
BAD_REQUEST
NOT_FOUND
FUND_NOT_FOUND
NOT_FUND_MEMBER
NOT_FUND_ADMIN
FUND_NOT_ACTIVE
ALREADY_MEMBER
INVALID_FUND_STATE
PAYMENT_NOT_FOUND
PAYMENT_ALREADY_PAID
USE_PROPOSAL
INTERNAL_ERROR
```
