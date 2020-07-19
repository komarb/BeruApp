package server

import (
	"beruAPI/models"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

func getItemsDimensions(items []models.Items) {
	for i, item := range items {
		err := db.Get(&items[i], "SELECT box_length, box_height, box_width, box_weight FROM products WHERE shop_sku=?", item.OfferID)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "sortItemsByVolume",
				"err" : err,
			},
			).Warn("Can't receive items dimensions!")
		}
	}
}

func makeMultipleProductBox(item *models.Items) {
	switch {
	case item.Length <= item.Height && item.Length <= item.Width:
		item.Length = item.Length * item.Count
	case item.Width <= item.Length && item.Width <= item.Height:
		item.Width = item.Width * item.Count
	case item.Height <= item.Length && item.Height <= item.Width:
		item.Height = item.Height * item.Count
	}
	item.Weight = item.Weight * float32(item.Count)
}

func volume(item models.Items) int {
	return item.Length * item.Width * item.Height
}

func addBoxToShipment(item models.Items, shipment *models.Shipment, orderID int64) {
	var tempBox models.Boxes
	tempBox.Width = item.Width
	tempBox.Height = item.Height
	tempBox.Depth = item.Length
	tempBox.Weight = int(item.Weight*1000)
	tempBox.FulfilmentID = fmt.Sprintf("%d-%d", orderID, len(shipment.Boxes)+1)
	shipment.Boxes = append(shipment.Boxes, tempBox)
}

func getIdFromMsg(msg string) string {
	return msg[6:]
}

func initPeriodicUpdate() {
	t := time.Now()
	n := time.Date(t.Year(), t.Month(), t.Day(), 19, 30, 0, 0, t.Location())
	d := n.Sub(t)
	if d < 0 {
		n = n.Add(24 * time.Hour)
		d = n.Sub(t)
	}
	for {
		time.Sleep(d)
		d = 24 * time.Hour
		UpdateStatusToShippedAll()
	}
}
