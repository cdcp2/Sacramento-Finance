package http

import (
	"github.com/gin-gonic/gin"
	"github.com/sacramento-finance/backend/internal/delivery/http/handler"
	"github.com/sacramento-finance/backend/internal/delivery/http/middleware"
)

type Handlers struct {
	Auth         *handler.AuthHandler
	User         *handler.UserHandler
	Fund         *handler.FundHandler
	Payment      *handler.PaymentHandler
	Proposal     *handler.ProposalHandler
	Dashboard    *handler.DashboardHandler
	Circulo      *handler.CirculoHandler
	Vaca         *handler.VacaHandler
	Fondo        *handler.FondoHandler
	Notification *handler.NotificationHandler
}

func SetupRouter(h *Handlers, jwtSecret string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	authRoutes := v1.Group("/auth")
	{
		authRoutes.POST("/register", h.Auth.Register)
		authRoutes.POST("/login", h.Auth.Login)
		authRoutes.POST("/refresh", h.Auth.Refresh)
	}

	protected := v1.Group("")
	protected.Use(middleware.Auth(jwtSecret))
	{
		protected.GET("/dashboard", h.Dashboard.Get)
		protected.GET("/users/me", h.User.Me)
		protected.PATCH("/users/me", h.User.UpdateMe)

		funds := protected.Group("/funds")
		{
			funds.POST("", h.Fund.Create)
			funds.GET("", h.Fund.ListMine)
			funds.GET("/:fund_id", h.Fund.Get)
			funds.PATCH("/:fund_id", h.Fund.UpdateRules)      // admin_only mode only
			funds.POST("/:fund_id/activate", h.Fund.Activate) // admin_only mode only
			funds.GET("/:fund_id/members", h.Fund.ListMembers)
			funds.POST("/:fund_id/members", h.Fund.AddMember)

			// Payments
			funds.GET("/:fund_id/payments", h.Payment.ListMine)
			funds.GET("/:fund_id/payments/all", h.Payment.ListAll)
			funds.POST("/:fund_id/payments/:payment_id/pay", h.Payment.Pay)
			funds.POST("/:fund_id/payments/:payment_id/waive", h.Payment.Waive) // admin_only mode only

			// Ledger
			funds.GET("/:fund_id/ledger", h.Payment.Ledger)

			// Governance — proposals and voting
			funds.POST("/:fund_id/proposals", h.Proposal.Create)
			funds.GET("/:fund_id/proposals", h.Proposal.List)
			funds.GET("/:fund_id/proposals/:proposal_id", h.Proposal.Get)
			funds.POST("/:fund_id/proposals/:proposal_id/vote", h.Proposal.Vote)

			// Circulo-specific
			funds.GET("/:fund_id/circulo", h.Circulo.GetConfig)
			funds.POST("/:fund_id/circulo/payout-order", h.Circulo.AssignPayoutOrder)
			funds.GET("/:fund_id/circulo/payouts", h.Circulo.ListPayouts)
			funds.POST("/:fund_id/circulo/close-round", h.Circulo.CloseRound)

			// Vaca-specific
			funds.GET("/:fund_id/vaca", h.Vaca.GetProgress)
			funds.POST("/:fund_id/vaca/distribute", h.Vaca.Distribute)

			// Fondo de ahorro-specific
			funds.GET("/:fund_id/fondo", h.Fondo.GetConfig)
			funds.GET("/:fund_id/fondo/my-balance", h.Fondo.GetMyBalance)
			funds.GET("/:fund_id/fondo/withdrawal-preview", h.Fondo.WithdrawalPreview)
			funds.POST("/:fund_id/fondo/accrue-interest", h.Fondo.AccrueInterest)
			funds.POST("/:fund_id/fondo/withdraw", h.Fondo.Withdraw)
		}

		// Notifications
		notifs := protected.Group("/notifications")
		{
			notifs.GET("", h.Notification.List)
			notifs.PATCH("/:notif_id/read", h.Notification.MarkRead)
			notifs.POST("/read-all", h.Notification.MarkAllRead)
		}
	}

	return r
}
