package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"ubex-be/models"

	"github.com/gorilla/mux"
)

type OrderHandler struct {
	DB *sql.DB
}

func NewOrderHandler(db *sql.DB) *OrderHandler {
	return &OrderHandler{DB: db}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		log.Printf("Transaction begin error: %v", err)
		return
	}
	defer tx.Rollback()

	currentCounter, err := getNextReceiptCounter(tx)
	if err != nil {
		http.Error(w, "Failed to get receipt counter", http.StatusInternalServerError)
		return
	}

	t := time.Now()
	formattedCounter := fmt.Sprintf("%04d", currentCounter)
	receiptNumber := fmt.Sprintf("UBX%04d%02d%02d%s",
		t.Year(), t.Month(), t.Day(), formattedCounter)

	order.ReceiptNumber = receiptNumber

	if order.ReceiptNumber == "" || order.AdminName == "" || order.SenderName == "" || order.ReceiverName == "" {
		http.Error(w, "Required fields are missing", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO orders (
			receipt_number, admin_name, sender_name, sender_phone, sender_address,
			sender_city, sender_district, sender_village, receiver_name, receiver_phone,
			receiver_address, receiver_city, receiver_district, receiver_village,
			package_content, package_weight, package_length, package_width, package_height,
			calculated_volume_weight, item_value, service_type, insurance_chosen, is_electronic,
			insurance_cost, payment_method, discount, total_shipping_cost, created_at, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		) RETURNING id, created_at`

	err = tx.QueryRow(
		query,
		order.ReceiptNumber, order.AdminName, order.SenderName, order.SenderPhone, order.SenderAddress,
		order.SenderCity, order.SenderDistrict, order.SenderVillage, order.ReceiverName, order.ReceiverPhone,
		order.ReceiverAddress, order.ReceiverCity, order.ReceiverDistrict, order.ReceiverVillage,
		order.PackageContent, order.PackageWeight, order.PackageLength, order.PackageWidth, order.PackageHeight,
		order.CalculatedVolumeWeight, order.ItemValue, order.ServiceType, order.InsuranceChosen, order.IsElectronic,
		order.InsuranceCost, order.PaymentMethod, order.Discount, order.TotalShippingCost, time.Now(), "pending",
	).Scan(&order.ID, &order.CreatedAt)

	if err != nil {
		log.Printf("Error inserting order: %v", err)
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		log.Printf("Commit error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func getNextReceiptCounter(tx *sql.Tx) (int64, error) {
	var currentCounter int64
	query := `SELECT current_counter FROM receipt_counters WHERE id = 1 FOR UPDATE`
	err := tx.QueryRow(query).Scan(&currentCounter)
	if err != nil {
		log.Printf("Failed to get counter: %v", err)
		return 0, err
	}

	newCounter := currentCounter + 1
	updateQuery := `UPDATE receipt_counters SET current_counter = $1 WHERE id = 1`
	_, err = tx.Exec(updateQuery, newCounter)
	if err != nil {
		log.Printf("Failed to update counter: %v", err)
		return 0, err
	}

	return newCounter, nil
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	adminName := r.URL.Query().Get("admin_name")

	query := "SELECT id, receipt_number, admin_name, sender_name, sender_phone, sender_address, sender_city, sender_district, sender_village, receiver_name, receiver_phone, receiver_address, receiver_city, receiver_district, receiver_village, package_content, package_weight, package_length, package_width, package_height, calculated_volume_weight, item_value, service_type, insurance_chosen, is_electronic, insurance_cost, payment_method, discount, total_shipping_cost, created_at, status  FROM orders WHERE 1=1"
	args := []interface{}{}
	argCounter := 1

	if startDateStr != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argCounter)
		args = append(args, startDateStr+" 00:00:00")
		argCounter++
	}
	if endDateStr != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argCounter)
		args = append(args, endDateStr+" 23:59:59")
		argCounter++
	}
	if adminName != "" {
		query += fmt.Sprintf(" AND admin_name ILIKE $%d", argCounter)
		args = append(args, "%"+adminName+"%")
		argCounter++
	}

	query += " ORDER BY created_at DESC"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var pkgLen, pkgWid, pkgHgt, calcVolWeight, itemVal sql.NullFloat64
		if err := rows.Scan(
			&order.ID, &order.ReceiptNumber, &order.AdminName, &order.SenderName, &order.SenderPhone, &order.SenderAddress,
			&order.SenderCity, &order.SenderDistrict, &order.SenderVillage, &order.ReceiverName, &order.ReceiverPhone,
			&order.ReceiverAddress, &order.ReceiverCity, &order.ReceiverDistrict, &order.ReceiverVillage,
			&order.PackageContent, &order.PackageWeight, &pkgLen, &pkgWid, &pkgHgt,
			&calcVolWeight, &itemVal, &order.ServiceType, &order.InsuranceChosen, &order.IsElectronic,
			&order.InsuranceCost, &order.PaymentMethod, &order.Discount, &order.TotalShippingCost, &order.CreatedAt, &order.Status,
		); err != nil {
			log.Printf("Error scanning order row: %v", err)
			http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
			return
		}

		if pkgLen.Valid {
			order.PackageLength = pkgLen.Float64
		}
		if pkgWid.Valid {
			order.PackageWidth = pkgWid.Float64
		}
		if pkgHgt.Valid {
			order.PackageHeight = pkgHgt.Float64
		}
		if calcVolWeight.Valid {
			order.CalculatedVolumeWeight = calcVolWeight.Float64
		}
		if itemVal.Valid {
			order.ItemValue = itemVal.Float64
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating over order rows: %v", err)
		http.Error(w, "Failed to retrieve orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// GetOrderSummary
func (h *OrderHandler) GetOrderSummary(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	adminName := r.URL.Query().Get("admin_name")

	query := `
        SELECT
			COALESCE(SUM(1), 0) AS total_orders,
			COALESCE(SUM(CASE WHEN status != 'cancelled' THEN total_shipping_cost ELSE 0 END), 0) AS total_revenue,
			COALESCE(SUM(CASE WHEN status != 'cancelled' AND payment_method = 'cash' THEN total_shipping_cost ELSE 0 END), 0) AS total_cash,
			COALESCE(SUM(CASE WHEN status != 'cancelled' AND payment_method = 'dfod' THEN total_shipping_cost ELSE 0 END), 0) AS total_dfod,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS total_pending,
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) AS total_cancelled
		FROM orders
		WHERE 1=1 `
	args := []interface{}{}
	argCounter := 1

	if startDateStr != "" {
		query += fmt.Sprintf(" AND created_at >= $%d", argCounter)
		args = append(args, startDateStr+" 00:00:00")
		argCounter++
	}
	if endDateStr != "" {
		query += fmt.Sprintf(" AND created_at <= $%d", argCounter)
		args = append(args, endDateStr+" 23:59:59")
		argCounter++
	}
	if adminName != "" {
		query += fmt.Sprintf(" AND admin_name ILIKE $%d", argCounter)
		args = append(args, "%"+adminName+"%")
		argCounter++
	}

	var totalOrders int
	var totalRevenue, totalCash, totalDfod float64
	var totalPending int
	var totalCancelled int

	err := h.DB.QueryRow(query, args...).Scan(&totalOrders, &totalRevenue, &totalCash, &totalDfod, &totalPending, &totalCancelled)
	if err != nil {
		log.Printf("Error querying order summary: %v", err)
		http.Error(w, "Failed to retrieve order summary", http.StatusInternalServerError)
		return
	}

	pendingPickup := totalPending

	summary := struct {
		TotalOrders    int     `json:"total_orders"`
		TotalRevenue   float64 `json:"total_revenue"`
		PendingPickup  int     `json:"pending_pickup"`
		TotalCash      float64 `json:"total_cash"`
		TotalDfod      float64 `json:"total_dfod"`
		TotalCancelled int     `json:"total_cancelled"`
	}{
		TotalOrders:    totalOrders,
		TotalRevenue:   totalRevenue,
		PendingPickup:  pendingPickup,
		TotalCash:      totalCash,
		TotalDfod:      totalDfod,
		TotalCancelled: totalCancelled,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	query := `UPDATE orders SET status = $1 WHERE id = $2 RETURNING receipt_number, status`
	var receiptNumber string
	var status string
	err = h.DB.QueryRow(query, "cancelled", id).Scan(&receiptNumber, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			log.Printf("Error canceling order %d: %v", id, err)
			http.Error(w, "Failed to cancel order", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":        fmt.Sprintf("Order %s successfully updated to status: %s", receiptNumber, status),
		"receipt_number": receiptNumber,
		"status":         status,
	})
}
