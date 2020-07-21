package server

import (
	"beruAPI/models"
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// sendStatus отправляет Беру информацию о новом статусе заказа по его ID
func sendStatus(status string, substatus string, orderID string) *http.Response {
	var orderStatus models.OrderStatusRequest
	orderStatus.Order.Status = status
	orderStatus.Order.Substatus = substatus
	json, _ := json.Marshal(orderStatus)
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%s/status.json", cfg.Beru.CampaignID, orderID)
	resp := DoAuthRequest("PUT", URL, bytes.NewBuffer(json))
	return resp
}

// sendStatus отправляет Беру информацию о нескольких заказах
func sendMultipleStatuses(orders models.MultipleOrderStatusRequest) *http.Response {
	json, _ := json.Marshal(orders)
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/status-update.json", cfg.Beru.CampaignID)
	resp := DoAuthRequest("PUT", URL, bytes.NewBuffer(json))
	return resp
}

// UpdateStatusShippedToAll устанавливает на все сегодняшние заказы со статусом "READY_TO_SHIP"
// статус "SHIPPED"
func UpdateStatusToShippedAll() {
	var readytoshipOrders models.OpenOrdersRequest
	var orders models.MultipleOrderStatusRequest
	var tempOrder models.MultipleOrderStatus
	date := time.Now().Format("02-01-2006")
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/" +
		"orders.json?status=PROCESSING&substatus=READY_TO_SHIP" +
		"&supplierShipmentDateFrom=%s&supplierShipmentDateTo=%s", cfg.Beru.CampaignID, date, date)
	resp := DoAuthRequest("GET", URL, nil)
	json.NewDecoder(resp.Body).Decode(&readytoshipOrders)

	for _, order := range readytoshipOrders.Orders {
		tempOrder.ID = order.ID
		tempOrder.Status = "PROCESSING"
		tempOrder.Substatus = "SHIPPED"
		orders.Orders = append(orders.Orders, tempOrder)
		if len(orders.Orders) == 30 {
			sendMultipleStatuses(orders)
			orders.Orders = nil
		}
	}
	if len(orders.Orders) != 0 {
		sendMultipleStatuses(orders)
	}
	log.Info("Statuses updates done successfully")
}
