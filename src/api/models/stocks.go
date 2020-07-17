package models

type StocksRequest struct {
	WarehouseID		int64		`json:"warehouseId"`
	Skus			[]string	`json:"skus"`
}
type StocksResponse struct {
	Skus []Skus `json:"skus"`
}
type StocksItems struct {
	Type      string    `json:"type"`
	Count     int64       `json:"count"`
	UpdatedAt string 	`json:"updatedAt"`
}
type Skus struct {
	Sku         string  `json:"sku"`
	WarehouseID int64     `json:"warehouseId"`
	Items       []StocksItems `json:"items"`
}