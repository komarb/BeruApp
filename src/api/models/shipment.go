package models

type Shipment struct {
	Boxes []Boxes `json:"boxes"`
}
type Boxes struct {
	FulfilmentID string `json:"fulfilmentId"`
	Weight       int    `json:"weight"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Depth        int    `json:"depth"`
	Items		 []BoxItem `json:"items,omitempty"`
}
type Dimensions struct {
	Length       int    `db:"box_length"`
	Width        int    `db:"box_width"`
	Height       int    `db:"box_height"`
}
type BoxItem struct {
	ID 			int64		`json:"id"`
	Count		int64		`json:"count"`
}
type Content struct {
	FulfilmentID string `db:"fulfilmentId"`
	OfferID string `db:"offerId"`
	Count   int	`db:"count"`
}