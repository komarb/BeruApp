package models

type OpenOrdersRequest struct {
	Orders []Order	`json:"orders"`
}
type AcceptOrderRequest struct {
	Order Order `json:"order"`
}
type Shipments struct {
	ID           int    `json:"id"`
	ShipmentDate string `json:"shipmentDate"`
}

type Delivery struct {
	RegionID    string      `json:"region_id"`
	Shipments   []Shipments `json:"shipments"`
	ServiceName string      `json:"serviceName"`
	Type        string      `json:"type"`
}

type Items struct {
	OfferID string `json:"offerId"`
	Count   int    `json:"count"`
	Price   float32    `json:"price"`
	Subsidy int    `json:"subsidy,omitempty"`
	Vat     string `json:"vat"`
	Length	int  `db:"box_length"`
	Width	int `db:"box_width"`
	Height	int `db:"box_height"`
	Weight  float32 `db:"box_weight"`
	Volume  int
}
type Order struct {
	Currency      string   `json:"currency"`
	Fake          bool     `json:"fake"`
	ID            int64      `json:"id"`
	ItemsTotal	  float32	`json:"itemsTotal"`
	Status		  string	`json:"status"`
	Substatus	  string	`json:"substatus"`
	PaymentType   string   `json:"paymentType"`
	PaymentMethod string   `json:"paymentMethod"`
	TaxSystem     string   `json:"taxSystem"`
	Total		  float32	`json:"total"`
	Delivery      Delivery `json:"delivery"`
	Items         []Items  `json:"items"`
}

type ReplyOrderRequest struct {
	Order ReplyOrder `json:"order"`
}
type ReplyOrder struct {
	Accepted bool   `json:"accepted"`
	ID       string `json:"id,omitempty"`
	Reason   string `json:"reason,omitempty"`
}
type OrderStatusRequest struct {
	Order		OrderStatus		`json:"order"`
}
type OrderStatus struct {
	Status		  string	`json:"status"`
	Substatus	  string	`json:"substatus"`
}
type MultipleOrderStatusRequest struct {
	Orders		[]MultipleOrderStatus		`json:"orders"`
}
type MultipleOrderStatus struct {
	ID			  int64	`json:"id"`
	Status		  string	`json:"status"`
	Substatus	  string	`json:"substatus"`
}