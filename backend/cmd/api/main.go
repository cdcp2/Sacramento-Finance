package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/sacramento-finance/backend/internal/config"
	deliveryhttp "github.com/sacramento-finance/backend/internal/delivery/http"
	"github.com/sacramento-finance/backend/internal/delivery/http/handler"
	"github.com/sacramento-finance/backend/internal/infrastructure/postgres"
	infranotif "github.com/sacramento-finance/backend/internal/infrastructure/notification"
	"github.com/sacramento-finance/backend/internal/infrastructure/repository"
	"github.com/sacramento-finance/backend/internal/usecase/auth"
	uccirculo "github.com/sacramento-finance/backend/internal/usecase/circulo"
	ucfondo "github.com/sacramento-finance/backend/internal/usecase/fondo"
	ucgovernance "github.com/sacramento-finance/backend/internal/usecase/governance"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
	ucpayment "github.com/sacramento-finance/backend/internal/usecase/payment"
	ucvaca "github.com/sacramento-finance/backend/internal/usecase/vaca"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	if cfg.Server.Mode == "release" {
		if cfg.Auth.JWTSecret == "change-me-in-production-please" {
			log.Fatal().Msg("AUTH_JWT_SECRET must be changed before running in release mode")
		}
		gin.SetMode(gin.ReleaseMode)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		gin.SetMode(gin.DebugMode)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	ctx := context.Background()
	db, err := postgres.NewPool(ctx, cfg.Database.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Str("host", cfg.Database.Host).Msg("connected to postgresql")

	// Repositories
	userRepo          := repository.NewUserRepo(db)
	fundRepo          := repository.NewFundRepo(db)
	memberRepo        := repository.NewMemberRepo(db)
	paymentRepo       := repository.NewPaymentRepo(db)
	ledgerRepo        := repository.NewLedgerRepo(db)
	proposalRepo      := repository.NewProposalRepo(db)
	voteRepo          := repository.NewVoteRepo(db)
	circuloRepo       := repository.NewCirculoConfigRepo(db)
	vacaRepo          := repository.NewVacaConfigRepo(db)
	fondoRepo         := repository.NewFondoConfigRepo(db)
	payoutRepo        := repository.NewPayoutRepo(db)
	notificationRepo  := repository.NewNotificationRepo(db)

	// Auth use cases
	accessTTL, err := time.ParseDuration(cfg.Auth.AccessTokenDuration)
	if err != nil {
		accessTTL = 15 * time.Minute
	}
	registerUC := auth.NewRegisterUseCase(userRepo)
	loginUC    := auth.NewLoginUseCase(userRepo, cfg.Auth.JWTSecret, accessTTL)
	refreshUC  := auth.NewRefreshUseCase(cfg.Auth.JWTSecret, accessTTL)

	// Payment use cases
	generateScheduleUC := ucpayment.NewGenerateScheduleUseCase(paymentRepo)
	recordPaymentUC    := ucpayment.NewRecordPaymentUseCase(paymentRepo, ledgerRepo)
	waivePaymentUC     := ucpayment.NewWaivePaymentUseCase(paymentRepo)
	markOverdueUC      := ucpayment.NewMarkOverdueUseCase(paymentRepo)

	// Product-specific use cases
	assignPayoutOrderUC := uccirculo.NewAssignPayoutOrderUseCase(circuloRepo, memberRepo)
	closeRoundUC        := uccirculo.NewCloseRoundUseCase(fundRepo, memberRepo, circuloRepo, payoutRepo, ledgerRepo)
	getProgressUC       := ucvaca.NewGetProgressUseCase(vacaRepo, ledgerRepo)
	distributeVacaUC    := ucvaca.NewDistributeUseCase(vacaRepo, fundRepo, memberRepo, ledgerRepo)
	accrueInterestUC    := ucfondo.NewAccrueInterestUseCase(fondoRepo, ledgerRepo)
	withdrawUC          := ucfondo.NewWithdrawUseCase(fondoRepo, ledgerRepo)

	// Governance use cases
	createProposalUC := ucgovernance.NewCreateProposalUseCase(proposalRepo, memberRepo)
	castVoteUC       := ucgovernance.NewCastVoteUseCase(
		proposalRepo, voteRepo,
		fundRepo, memberRepo,
		paymentRepo, generateScheduleUC, distributeVacaUC,
	)

	// Notification service (with optional email sender)
	notifSvc := ucnotif.NewService(notificationRepo)
	if cfg.Email.Enabled {
		smtpSender := infranotif.NewSMTPSender(
			cfg.Email.Host, cfg.Email.Port,
			cfg.Email.Username, cfg.Email.Password, cfg.Email.From,
		)
		notifSvc = notifSvc.WithEmail(smtpSender, userRepo)
		log.Info().Str("host", cfg.Email.Host).Msg("email sender enabled")
	}

	// Handlers
	authHandler         := handler.NewAuthHandler(registerUC, loginUC, refreshUC)
	userHandler         := handler.NewUserHandler(userRepo)
	fundHandler         := handler.NewFundHandler(fundRepo, memberRepo, userRepo, generateScheduleUC, circuloRepo, vacaRepo, fondoRepo, notifSvc)
	paymentHandler      := handler.NewPaymentHandler(recordPaymentUC, waivePaymentUC, paymentRepo, ledgerRepo, fundRepo, memberRepo, notifSvc)
	proposalHandler     := handler.NewProposalHandler(createProposalUC, castVoteUC, proposalRepo, voteRepo, fundRepo, memberRepo, notifSvc)
	dashboardHandler    := handler.NewDashboardHandler(fundRepo, memberRepo, paymentRepo, proposalRepo, notificationRepo)
	circuloHandler      := handler.NewCirculoHandler(assignPayoutOrderUC, closeRoundUC, circuloRepo, payoutRepo, fundRepo, memberRepo, notifSvc)
	vacaHandler         := handler.NewVacaHandler(getProgressUC, distributeVacaUC, vacaRepo, fundRepo, memberRepo, notifSvc)
	fondoHandler        := handler.NewFondoHandler(accrueInterestUC, withdrawUC, fondoRepo, ledgerRepo, fundRepo, memberRepo, notifSvc)
	notificationHandler := handler.NewNotificationHandler(notificationRepo)

	router := deliveryhttp.SetupRouter(&deliveryhttp.Handlers{
		Auth:         authHandler,
		User:         userHandler,
		Fund:         fundHandler,
		Payment:      paymentHandler,
		Proposal:     proposalHandler,
		Dashboard:    dashboardHandler,
		Circulo:      circuloHandler,
		Vaca:         vacaHandler,
		Fondo:        fondoHandler,
		Notification: notificationHandler,
	}, cfg.Auth.JWTSecret)

	// Background job: mark overdue payments once per day at startup + every 24h
	go func() {
		runMarkOverdue := func() {
			n, err := markOverdueUC.Execute(context.Background(), time.Now().UTC())
			if err != nil {
				log.Error().Err(err).Msg("mark overdue job failed")
				return
			}
			if n > 0 {
				log.Info().Int("count", n).Msg("marked overdue payments")
			}
		}
		runMarkOverdue() // run once at startup
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runMarkOverdue()
		}
	}()

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server shutdown error")
	}
	log.Info().Msg("server stopped")
}
