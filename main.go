package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"ubex-be/database"
	"ubex-be/handlers"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	r := mux.NewRouter()

	orderHandler := handlers.NewOrderHandler(db)

	r.HandleFunc("/api/orders", orderHandler.CreateOrder).Methods("POST")
	r.HandleFunc("/api/orders", orderHandler.GetOrders).Methods("GET")
	r.HandleFunc("/api/summary", orderHandler.GetOrderSummary).Methods("GET")
	r.HandleFunc("/api/orders/{id}/cancel", orderHandler.CancelOrder).Methods("PUT")
	//r.HandleFunc("/api/next-receipt-counter", orderHandler.GetNextReceiptCounterHandler).Methods("POST")
	// r.HandleFunc("/api/orders/{id}", orderHandler.UpdateOrder).Methods("PUT")
	// r.HandleFunc("/api/orders/{id}", orderHandler.DeleteOrder).Methods("DELETE")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // allow buat semua origin(masa development)
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serverAddr := ":" + port
	log.Printf("Server starting on %s...", serverAddr)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
