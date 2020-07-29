package server

import (
	"beruAPI/models"
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sort"
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
	resp := DoAuthRequest("POST", URL, bytes.NewBuffer(json))
	return resp
}

// UpdateStatusShippedToAll устанавливает на все сегодняшние заказы со статусом "READY_TO_SHIP"
// статус "SHIPPED"
func UpdateStatusToShippedAll() {
	var readytoshipOrders models.OpenOrdersRequest
	var orders models.MultipleOrderStatusRequest
	var tempOrder models.MultipleOrderStatus
	var reply models.OrderStatusReply
	dateNow := time.Now().Format("02-01-2006")
	dateFrom := time.Now().AddDate(0,0,-1).Format("02-01-2006")
	dateTo := time.Now().AddDate(0,0,1).Format("02-01-2006")
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/" +
		"orders.json?status=PROCESSING&substatus=READY_TO_SHIP" +
		"&supplierShipmentDateFrom=%s&supplierShipmentDateTo=%s", cfg.Beru.CampaignID, dateFrom, dateTo)
	resp := DoAuthRequest("GET", URL, nil)
	json.NewDecoder(resp.Body).Decode(&readytoshipOrders)
	for _, order := range readytoshipOrders.Orders {
		if order.Delivery.Shipments[0].ShipmentDate == dateNow {
			tempOrder.ID = order.ID
			tempOrder.Status = "PROCESSING"
			tempOrder.Substatus = "SHIPPED"
			orders.Orders = append(orders.Orders, tempOrder)
			if len(orders.Orders) == 30 {
				resp := sendMultipleStatuses(orders)
				json.NewDecoder(resp.Body).Decode(&reply)
				for _, orderReply := range reply.Orders {
					if orderReply.UpdateStatus == "ERROR" {
						log.WithFields(log.Fields{
							"orderId" : orderReply.ID,
							"error":    orderReply.ErrorDetails},
						).Warn("Order status wasn't set to SHIPPED!")
					}
				}
				orders.Orders = nil
			}
		}
	}
	if len(orders.Orders) != 0 {
		resp := sendMultipleStatuses(orders)
		json.NewDecoder(resp.Body).Decode(&reply)
		for _, orderReply := range reply.Orders {
			if orderReply.UpdateStatus == "ERROR" {
				log.WithFields(log.Fields{
					"orderId" : orderReply.ID,
					"error":    orderReply.ErrorDetails},
				).Warn("Order status wasn't set to SHIPPED!")
			}
		}
	}
}

// sendShipmentsInfo создает грузовые места по заказу и отправляет информацию серверам Беру
func sendShipmentsInfo(inputOrder models.AcceptOrderRequest) {
	var shipment models.Shipment
	boxs := inputOrder.Order.Items
	getItemsDimensions(boxs)
	for _, box := range boxs {
		if box.Count > 1 && volume(box) < 1100 {
			makeMultipleProductBox(&box)
			addBoxToShipment(box, &shipment, inputOrder.Order.ID)
		} else {
			for j := 0; j < box.Count; j++ {
				tempBox := box
				tempBox.Count = 1
				addBoxToShipment(tempBox, &shipment, inputOrder.Order.ID)
			}
		}
	}
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%d/delivery/shipments/%d/boxes.json",
		cfg.Beru.CampaignID, inputOrder.Order.ID, inputOrder.Order.Delivery.Shipments[0].ID)

	DoAuthRequestWithObj("PUT", URL, shipment)
}

// getOpenOrders запрашивает у Беру информацию о всех открытых (со статусом PROCESSING) заказах
func getOpenOrders() string {
	var inputOrders models.OpenOrdersRequest
	var resultMsg string
	var ordersMsg string
	var currentDate string
	openOrdersCount := 0
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders.json?status=PROCESSING", cfg.Beru.CampaignID)
	resp := DoAuthRequest("GET", URL, nil)

	json.NewDecoder(resp.Body).Decode(&inputOrders)
	sort.Slice(inputOrders.Orders, func(i, j int) bool {
		date1 := inputOrders.Orders[i].Delivery.Shipments[0].ShipmentDate
		date2 := inputOrders.Orders[j].Delivery.Shipments[0].ShipmentDate
		return date1 < date2
	})
	for _, order := range inputOrders.Orders {
		if order.CancelRequested || !cfg.App.TestMode && order.Fake{
			continue
		}
		if currentDate != order.Delivery.Shipments[0].ShipmentDate {
			currentDate = order.Delivery.Shipments[0].ShipmentDate
			ordersMsg += fmt.Sprintf("------------------------------------------------\n▪️ *%s*\n", currentDate)
		}
		ordersMsg += fmt.Sprintf("_Заказ №%d_\n`%s`\n", order.ID, order.Substatus)
		for _, item := range order.Items {
			ordersMsg += fmt.Sprintf("`%s`\n`%d` шт.\n",
					item.OfferID, item.Count)
		}
		ordersMsg += fmt.Sprintf("Подробнее: /order%d\n\n", order.ID)
		openOrdersCount += 1
	}
	resultMsg += fmt.Sprintf("Всего заказов: `%d`\n\n", openOrdersCount)
	resultMsg += ordersMsg
	return resultMsg
}

// getOpenOrders запрашивает у Беру информацию об определнном заказе по его ID
func getOrderInfo(orderID string) string {
	var inputOrder models.AcceptOrderRequest
	var content models.Content
	var msgText string
	orderURL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%s.json", cfg.Beru.CampaignID, orderID)
	resp := DoAuthRequest("GET", orderURL, nil)
	json.NewDecoder(resp.Body).Decode(&inputOrder)
	if resp.StatusCode == 404 || resp.StatusCode == 403 || inputOrder.Order.ID == 0 {
		msgText = "Заказ с таким ID *не найден*"
	} else {
		msgText = fmt.Sprintf("*Заказ №%d:*\n", inputOrder.Order.ID)
		msgText += fmt.Sprintf("*Статус заказа:* `%s`\n*Субстатус:* `%s`\n---------------------\n", inputOrder.Order.Status, inputOrder.Order.Substatus)
		for _, box := range inputOrder.Order.Delivery.Shipments[0].Boxes {
			msgText += fmt.Sprintf("+ _Грузовое место №%s:_\n`%.2f` кг `%d/%d/%d`\n",
				box.FulfilmentID, float32(box.Weight)/1000, box.Height, box.Width, box.Depth)
			db.Get(&content, "SELECT * FROM shipments WHERE fulfilmentId=?", box.FulfilmentID)
			for _, item := range inputOrder.Order.Items {
				if item.OfferID == content.OfferID {
					msgText += fmt.Sprintf("`%s`\n`%d` шт.\n`%.2f` ₽\n",
						item.OfferID, content.Count, float32(content.Count) * item.Price)
					break
				}
			}
		}
		msgText += fmt.Sprintf("---------------------\n`%.2f` ₽\n", inputOrder.Order.ItemsTotal)
		if inputOrder.Order.PaymentType == "PREPAID" {
			msgText += "*Оплата:*\n`Оплачен`\n"
		} else {
			msgText += "*Оплата:*`Оплата при получении, "
			switch inputOrder.Order.PaymentMethod {
			case "CARD_ON_DELIVERY":
				msgText += "банковской картой`\n"
			case "CASH_ON_DELIVERY":
				msgText += "наличными`\n"
			}
		}
		msgText += fmt.Sprintf("*Доставка:*\n`№%d`\n`%s`\n", inputOrder.Order.Delivery.Shipments[0].ID, inputOrder.Order.Delivery.Shipments[0].ShipmentDate)
		msgText += fmt.Sprintf("*Ярлыки-наклейки:\n* /label%d", inputOrder.Order.ID)
	}
	// Кнопка отмены заказа
	/*if inputOrder.Order.Substatus == "STARTED" || inputOrder.Order.Substatus == "READY_TO_SHIP" {
		msg.ReplyMarkup = orderControlKeyboard
	}*/
	return msgText
}