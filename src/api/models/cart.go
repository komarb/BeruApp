package models

type RelevantCartRequest struct {
	Cart			Cart			`json:"cart"`
}
type Cart struct {
	Items			[]Item   		`json:"items"`
}
type Item struct {
	FeedId			uint64			`json:"feedId"`
	OfferId			string			`json:"offerId" db:"offerId"`
	Count 			int				`json:"count" db:"count"`
}
