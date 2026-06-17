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
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

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

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(&cfg.JWT)

	// Initialize payment service
	paymentService := services.NewPaymentService(db, lndService, exchangeService, cfg)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db, cfg, authMiddleware)
	paymentHandler := handlers.NewPaymentHandler(paymentService, exchangeService)

	// Setup Gin router
	router := setupRouter(cfg, authMiddleware, userHandler, paymentHandler)

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
	sqlDB.Close()
	log.Println("Server stopped")
}

func setupRouter(
	cfg *config.Config,
	auth *middleware.AuthMiddleware,
	userHandler *handlers.UserHandler,
	paymentHandler *handlers.PaymentHandler,
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

		// Webhook (no auth - secured by signature verification in production)
		v1.POST("/payments/webhook", paymentHandler.Webhook)
	}

	return router
}