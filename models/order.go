package models

import "time"

type Order struct {
	ID            int    `json:"id"`
	ReceiptNumber string `json:"receipt_number"`
	AdminName     string `json:"admin_name"`
	
	SenderName     string `json:"sender_name"`
	SenderPhone    string `json:"sender_phone"`
	SenderAddress  string `json:"sender_address"`
	SenderCity     string `json:"sender_city"`
	SenderDistrict string `json:"sender_district"`
	SenderVillage  string `json:"sender_village"`

	ReceiverName     string `json:"receiver_name"`
	ReceiverPhone    string `json:"receiver_phone"`
	ReceiverAddress  string `json:"receiver_address"`
	ReceiverCity     string `json:"receiver_city"`
	ReceiverDistrict string `json:"receiver_district"`
	ReceiverVillage  string `json:"receiver_village"`

	PackageContent         string  `json:"package_content"`
	PackageWeight          float64 `json:"package_weight"`
	PackageLength          float64 `json:"package_length"`
	PackageWidth           float64 `json:"package_width"`
	PackageHeight          float64 `json:"package_height"`
	CalculatedVolumeWeight float64 `json:"calculated_volume_weight"`

	ItemValue         float64 `json:"item_value"`
	ServiceType       string  `json:"service_type"`
	InsuranceChosen   bool    `json:"insurance_chosen"`
	IsElectronic      bool    `json:"is_electronic"`
	InsuranceCost     float64 `json:"insurance_cost"`
	PaymentMethod     string  `json:"payment_method"`
	Discount          float64 `json:"discount"`
	TotalShippingCost float64 `json:"total_shipping_cost"`

	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type OrderFilterParams struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	AdminName string `json:"admin_name"`
	Status    string `json:"status"`
}

type ReceiptCounter struct {
	ID             int       `json:"id"`
	CurrentCounter int64     `json:"current_counter"`
	LastResetDate  time.Time `json:"last_reset_date"`
}
