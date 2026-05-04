/**
 * E2E: invitation flow
 *
 * Scenario A — accept:
 *   1. Admin registers, creates a fund, invites Guest by email.
 *   2. Guest logs in, goes to Notifications, accepts the invitation.
 *   3. The invitation card flips to "Aceptada".
 *   4. Guest can see the fund in their list.
 *
 * Scenario B — reject:
 *   Uses the same admin fund; Guest rejects a second invitation.
 *   The card flips to "Rechazada".
 *
 * Setup: users and the fund are created via API calls (no browser),
 * then we drive the full UI for the invitation interaction itself.
 */
import { expect, test } from '@playwright/test'
import { createFund, loginUser, registerUser, type TestFund, type TestUser } from './helpers/api'

// ── Shared state (populated in beforeAll) ─────────────────────────────────────

let admin: TestUser
let guest: TestUser
let fund: TestFund

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Fills credentials in the login form and waits for the dashboard. */
async function loginViaUI(
  page: import('@playwright/test').Page,
  email: string,
  password: string,
) {
  await page.goto('/login')
  await page.getByLabel('Correo electrónico').fill(email)
  await page.getByLabel('Contraseña').fill(password)
  await page.getByRole('button', { name: 'Ingresar' }).click()
  await expect(page).toHaveURL(/\/app\/dashboard/, { timeout: 10_000 })
}

/** Clicks logout via the profile page and waits for the login screen. */
async function logoutViaUI(page: import('@playwright/test').Page) {
  await page.goto('/app/profile')
  await page.getByRole('button', { name: 'Cerrar sesión' }).click()
  await expect(page).toHaveURL(/\/login/, { timeout: 8_000 })
}

// ── Setup ─────────────────────────────────────────────────────────────────────

test.beforeAll(async ({ request }) => {
  // Create two independent users via API — avoids polluting the DB with UI signups
  admin = await registerUser(request, 'admin')
  guest = await registerUser(request, 'guest')

  // Admin creates the fund they'll invite the guest into
  const token = await loginUser(request, admin.email, admin.password)
  fund = await createFund(request, token)
})

// ── Scenario A: Guest accepts the invitation ──────────────────────────────────

test('admin invites guest by email — guest accepts and joins the fund', async ({ page }) => {
  // ── 1. Admin logs in ───────────────────────────────────────────────────────
  await loginViaUI(page, admin.email, admin.password)

  // ── 2. Admin goes to fund detail and invites guest ─────────────────────────
  await page.goto(`/app/funds/${fund.id}`)
  await expect(page.getByRole('heading', { name: fund.name })).toBeVisible()

  // The invite form defaults to "Correo" — just fill the field and submit
  await page.getByLabel('Correo del usuario').fill(guest.email)
  await page.getByRole('button', { name: 'Enviar invitación' }).click()

  await expect(page.getByText('Invitación enviada correctamente')).toBeVisible()

  // The invitation list in the "Invitaciones" section now shows one pending entry
  await expect(page.getByText('Pendiente').first()).toBeVisible()

  // ── 3. Admin logs out ──────────────────────────────────────────────────────
  await logoutViaUI(page)

  // ── 4. Guest logs in ───────────────────────────────────────────────────────
  await loginViaUI(page, guest.email, guest.password)

  // ── 5. Guest goes to Notifications and accepts ─────────────────────────────
  await page.goto('/app/notifications')

  // The "Invitaciones pendientes" banner must be visible with a count badge
  await expect(page.getByText('Invitaciones pendientes')).toBeVisible()
  // Count badge shows "1"
  await expect(page.getByText('1').first()).toBeVisible()

  // Click the accept button on the first (only) pending invitation
  await page.getByRole('button', { name: 'Aceptar' }).click()

  // ── 6. Verify the invitation is now accepted ───────────────────────────────
  // Pending section disappears; the item moves to "Invitaciones anteriores"
  await expect(page.getByText('Invitaciones pendientes')).not.toBeVisible({ timeout: 6_000 })
  await expect(page.getByText('Invitaciones anteriores')).toBeVisible()
  await expect(page.getByText('Aceptada')).toBeVisible()

  // ── 7. Guest can now see the fund in their fund list ───────────────────────
  await page.goto('/app/funds')
  await expect(page.getByText(fund.name)).toBeVisible()
})

// ── Scenario B: Guest rejects the invitation ──────────────────────────────────

test('admin invites guest again — guest rejects', async ({ request, page }) => {
  // Guest was already accepted into the fund in Scenario A.
  // We need a *new* fund so the invitation isn't blocked by "ALREADY_MEMBER".
  const token = await loginUser(request, admin.email, admin.password)
  const secondFund = await createFund(request, token)

  // ── 1. Admin sends invitation to the second fund ───────────────────────────
  await loginViaUI(page, admin.email, admin.password)
  await page.goto(`/app/funds/${secondFund.id}`)
  await expect(page.getByRole('heading', { name: secondFund.name })).toBeVisible()

  await page.getByLabel('Correo del usuario').fill(guest.email)
  await page.getByRole('button', { name: 'Enviar invitación' }).click()
  await expect(page.getByText('Invitación enviada correctamente')).toBeVisible()

  await logoutViaUI(page)

  // ── 2. Guest logs in and rejects ──────────────────────────────────────────
  await loginViaUI(page, guest.email, guest.password)
  await page.goto('/app/notifications')

  await expect(page.getByText('Invitaciones pendientes')).toBeVisible()
  await page.getByRole('button', { name: 'Rechazar' }).click()

  // Card flips to "Rechazada" in "Invitaciones anteriores"
  await expect(page.getByText('Invitaciones pendientes')).not.toBeVisible({ timeout: 6_000 })
  await expect(page.getByText('Rechazada')).toBeVisible()

  // The second fund must NOT appear in the guest's fund list
  await page.goto('/app/funds')
  await expect(page.getByText(secondFund.name)).not.toBeVisible()
})
