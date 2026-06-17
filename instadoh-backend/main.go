package main

import (
	"log"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"instadoh-backend/config"
	"instadoh-backend/database"
	"instadoh-backend/handlers"
	"instadoh-backend/middleware"
	"instadoh-backend/services"

	"github.com/gin-gonic/gin"
	cors "github.com/gin-contrib/cors"
)

func main() {
	log.Println("Starting InstaDoh - Instant cross-border Lightning payments")

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db := database.Init(cfg)
	sqlDB, dbErr := db.DB()
	if dbErr == nil {
		defer sqlDB.Close()
	}

	// Initialize LND service
	lndService := services.NewLNDService(&cfg.LND)
	if !lndService.IsConnected() {
		log.Println("WARNING: Running without LND connection. Payment operations will be limited.")
	}

	// Initialize exchange rate service
	exchangeService := services.NewExchangeService(
		cfg.Exchange.BaseURL,
		cfg.Exchange.APIKey,
		cfg.Exchange.CacheTTL,
	)

	// Initialize M-Pesa service
	mpesaConfig := &services.MpesaConfig{
		ConsumerKey:    cfg.Mpesa.ConsumerKey,
		ConsumerSecret: cfg.Mpesa.ConsumerSecret,
		Passkey:        cfg.Mpesa.Passkey,
		ShortCode:      cfg.Mpesa.ShortCode,
		Environment:    cfg.Mpesa.Environment,
		CallbackURL:    cfg.Mpesa.CallbackURL,
	}
	mpesaService := services.NewMpesaService(mpesaConfig)
	if cfg.Mpesa.ConsumerKey == "" {
		log.Println("WARNING: M-Pesa not configured (missing MPESA_CONSUMER_KEY). Kenyan mobile money features disabled.")
	} else {
		log.Println("M-Pesa service initialized")
	}

	// Initialize Uganda Mobile Money service
	ugandaMobileConfig := &services.UgandaMobileConfig{
		APIBaseURL:  cfg.UgandaMobile.APIBaseURL,
		APIUsername: cfg.UgandaMobile.APIUsername,
		APIPassword: cfg.UgandaMobile.APIPassword,
		CallbackURL: cfg.UgandaMobile.CallbackURL,
	}
	ugandaMobileService := services.NewUgandaMobileService(ugandaMobileConfig)
	log.Println("Uganda Mobile Money service initialized")

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(&cfg.JWT)

	// Initialize payment service
	paymentService := services.NewPaymentService(db, lndService, exchangeService, cfg)

	// Initialize cross-border service
	crossBorderService := services.NewCrossBorderService(
		db,
		exchangeService,
		lndService,
		mpesaService,
		ugandaMobileService,
	)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db, cfg, authMiddleware)
	paymentHandler := handlers.NewPaymentHandler(paymentService, exchangeService)
	crossBorderHandler := handlers.NewCrossBorderHandler(crossBorderService)

	// Setup Gin router
	router := setupRouter(cfg, authMiddleware, userHandler, paymentHandler, crossBorderHandler)

	// Start server
	addr := cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if dbErr == nil && sqlDB != nil {
		sqlDB.Close()
	}
	log.Println("Server stopped")
}

func setupRouter(
	cfg *config.Config,
	auth *middleware.AuthMiddleware,
	userHandler *handlers.UserHandler,
	paymentHandler *handlers.PaymentHandler,
	crossBorderHandler *handlers.CrossBorderHandler,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "1.0.0",
			"name":    "InstaDoh API",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		v1.GET("/countries", userHandler.GetCountries)
		v1.GET("/payments/rates", paymentHandler.GetExchangeRate)

		// Auth routes
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", userHandler.Register)
			authGroup.POST("/login", userHandler.Login)
			authGroup.POST("/refresh", auth.RequireAuth(), userHandler.RefreshToken)
		}

		// Protected user routes
		userGroup := v1.Group("/users")
		userGroup.Use(auth.RequireAuth())
		{
			userGroup.GET("/me", userHandler.GetProfile)
		}

		// Protected payment routes
		paymentGroup := v1.Group("/payments")
		paymentGroup.Use(auth.RequireAuth())
		{
			paymentGroup.POST("/invoices", paymentHandler.CreateInvoice)
			paymentGroup.POST("/send", paymentHandler.SendPayment)
			paymentGroup.GET("/transactions", paymentHandler.ListTransactions)
			paymentGroup.GET("/transactions/:id", paymentHandler.GetTransaction)
			paymentGroup.GET("/balance", paymentHandler.GetBalance)
			paymentGroup.GET("/stats", paymentHandler.GetStats)
			paymentGroup.POST("/convert", paymentHandler.ConvertCurrency)
		}

		// Cross-border payment routes (protected)
		crossBorderGroup := v1.Group("/cross-border")
		crossBorderGroup.Use(auth.RequireAuth())
		{
			crossBorderGroup.GET("/quote", crossBorderHandler.GetQuote)
			crossBorderGroup.POST("/send", crossBorderHandler.SendCrossBorder)
			crossBorderGroup.GET("/transactions", crossBorderHandler.ListCrossBorderTransactions)
			crossBorderGroup.GET("/transactions/:id", crossBorderHandler.GetCrossBorderTransaction)
		}

		// M-Pesa mobile money routes (Kenya)
		mpesaGroup := v1.Group("/mpesa")
		{
			// Protected endpoints
			mpesaProtected := mpesaGroup.Group("")
			mpesaProtected.Use(auth.RequireAuth())
			{
				mpesaProtected.POST("/deposit", crossBorderHandler.DepositMpesa)
				mpesaProtected.POST("/withdraw", crossBorderHandler.WithdrawMpesa)
			}
			// Public webhook/callback routes (secured by signature verification in production)
			mpesaGroup.POST("/callback", crossBorderHandler.MpesaCallback)
			mpesaGroup.POST("/result", crossBorderHandler.MpesaResultCallback)
			mpesaGroup.POST("/timeout", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Success"})
			})
		}

		// Uganda Mobile Money routes
		ugandaMobileGroup := v1.Group("/uganda-mobile")
		{
			ugandaMobileProtected := ugandaMobileGroup.Group("")
			ugandaMobileProtected.Use(auth.RequireAuth())
			{
				ugandaMobileProtected.POST("/deposit", crossBorderHandler.DepositUgandaMobile)
				ugandaMobileProtected.POST("/withdraw", crossBorderHandler.WithdrawUgandaMobile)
			}
			ugandaMobileGroup.POST("/callback", crossBorderHandler.UgandaMobileCallback)
		}

		// Webhook (no auth - secured by signature verification in production)
		v1.POST("/payments/webhook", paymentHandler.Webhook)
	}

	return router
}