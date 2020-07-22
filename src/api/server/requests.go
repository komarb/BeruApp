package server

import (
	"beruAPI/models"
	"bytes"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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

// sendShipmentsInfo создает грузовые места по заказу и отправляет информацию серверам Беру
func sendShipmentsInfo(inputOrder models.AcceptOrderRequest) {
	var shipment models.Shipment
	items := inputOrder.Order.Items
	getItemsDimensions(items)
	for _, item := range items {
		if item.Count > 1 && volume(item) < 1100 {
			makeMultipleProductBox(&item)
			addBoxToShipment(item, &shipment, inputOrder.Order.ID)
		} else {
			for j := 0; j < item.Count; j++ {
				addBoxToShipment(item, &shipment, inputOrder.Order.ID)
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
	URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders.json?status=PROCESSING", cfg.Beru.CampaignID)
	resp := DoAuthRequest("GET", URL, nil)

	json.NewDecoder(resp.Body).Decode(&inputOrders)
	resultMsg += fmt.Sprintf("Всего заказов: %d\n\n", len(inputOrders.Orders))
	for _, order := range inputOrders.Orders {
		if order.CancelRequested {
			continue
		}
		resultMsg += fmt.Sprintf("*Заказ №%d*\n*Статус:* %s, *субстатус:* %s\nПодробнее о заказе: /order%d\n\n-----", order.ID, order.Status, order.Substatus, order.ID)
	}
	return resultMsg
}

// getOpenOrders запрашивает у Беру информацию об определнном заказе по его ID
func getOrderInfo(orderID string, chatID int64) {
	var inputOrder models.AcceptOrderRequest
	var msgText string
	orderURL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%s.json", cfg.Beru.CampaignID, orderID)
	resp := DoAuthRequest("GET", orderURL, nil)
	json.NewDecoder(resp.Body).Decode(&inputOrder)
	if resp.StatusCode == 404 || resp.StatusCode == 403 || inputOrder.Order.ID == 0{
		msgText = "Заказ с таким ID *не найден*"
	} else {
		msgText := fmt.Sprintf("*Заказ №%d:*\n", inputOrder.Order.ID)
		msgText += fmt.Sprintf("*Статус заказа:* %s, субстатус: %s\n\n-----", inputOrder.Order.Status, inputOrder.Order.Substatus)
		for i, item := range inputOrder.Order.Items {
			msgText += fmt.Sprintf("_Товар №%d:_\nOfferID товара: `%s`\nКоличество: `%d`\nЦена за шутку: `%.2f`\nРазмеры(длина, ширина, высота в см): `%d/%d/%d`\nВес (в кг): `%.2f`",
				i+1, item.OfferID, item.Count, item.Price, item.Length, item.Width, item.Height, item.Weight)
		}
		msgText += fmt.Sprintf("*Общая стоимость товаров (не включая доставку): `%f`*", inputOrder.Order.ItemsTotal)
		msgText += fmt.Sprintf("*Информация о доставке:*\n\n-----ID посылки: `%d`\nДата отгрузки: `%s`\n", inputOrder.Order.Delivery.Shipments[0].ID, inputOrder.Order.Delivery.Shipments[0].ShipmentDate)
		msgText += fmt.Sprintf("*Ссылка на скачивание ярлыков-наклеек на грузовые места:*\n/label%d", inputOrder.Order.ID)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "markdown"

	// Кнопка отмены заказа
	/*if inputOrder.Order.Substatus == "STARTED" || inputOrder.Order.Substatus == "READY_TO_SHIP" {
		msg.ReplyMarkup = orderControlKeyboard
	}*/
	bot.Send(msg)
}