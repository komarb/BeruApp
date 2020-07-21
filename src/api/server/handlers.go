package server

import (
	"beruAPI/models"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
)

// getItemsRelevantInfo отвечает на запрос со списком товаров релевантной информацией
// об их количестве
func getItemsRelevantInfo(w http.ResponseWriter, r *http.Request) {
	var inputCart models.RelevantCartRequest
	var relevantItem models.Item
	var relevantCart models.RelevantCartRequest
	count := 0
	notInStock := 0
	json.NewDecoder(r.Body).Decode(&inputCart)
	for _, item := range inputCart.Cart.Items {
		err := db.Get(&count, "SELECT count FROM products WHERE shop_sku=?", item.OfferId)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "db.Get",
				"offerId" : item.OfferId,
			},
			).Warn("OfferId wasn't found in database, set count to 0")
			count = 0
		}
		relevantItem.FeedId = item.FeedId
		relevantItem.OfferId = item.OfferId
		relevantItem.Count = count
		relevantCart.Cart.Items = append(relevantCart.Cart.Items, relevantItem)
		if count == 0 {
			notInStock++
		}
	}
	if notInStock == len(inputCart.Cart.Items) {
		relevantCart.Cart.Items = []models.Item{}
	}
	json.NewEncoder(w).Encode(relevantCart)
}

// getOrderAcceptanceStatus отвечает на запрос с новым заказом - если в наличии имеется
// достаточное количество товара, то заказ будет принят, иначе отклонен
func getOrderAcceptanceStatus(w http.ResponseWriter, r *http.Request) {
	var inputOrder models.AcceptOrderRequest
	var replyOrder models.ReplyOrderRequest
	var acceptOrder bool
	count := 0
	json.NewDecoder(r.Body).Decode(&inputOrder)
	for _, item := range inputOrder.Order.Items {
		log.Info(item.OfferID)
		err := db.Get(&count, "SELECT count FROM products WHERE shop_sku=?", item.OfferID)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "db.Get",
				"handler" : "getOrderAcceptanceStatus",
				"offerId" : item.OfferID,
			},
			).Warn("Something went wrong while selecting, declining an order")
			acceptOrder = false
		}
		if count < item.Count {
			log.Info("Not enough items in stock, declining order №" + strconv.FormatInt(inputOrder.Order.ID, 10))
			acceptOrder = false
			break
		} else {
			acceptOrder = true
		}
	}
	if acceptOrder {
		replyOrder.Order.ID = strconv.FormatInt(inputOrder.Order.ID, 10)
		replyOrder.Order.Accepted = true

		msg := fmt.Sprintf("*Новый заказ №%d:*\n", inputOrder.Order.ID)
		for i, item := range inputOrder.Order.Items {
			msg += fmt.Sprintf("_Товар №%d:_\nOfferID товара: `%s`\nКоличество: `%d`\nЦена за шутку: `%.2f`\n", i+1, item.OfferID, item.Count, item.Price)
		}
		msg += fmt.Sprintf("*Информация о доставке:*\nID посылки: `%d`\nДата отгрузки: `%s`\n", inputOrder.Order.Delivery.Shipments[0].ID, inputOrder.Order.Delivery.Shipments[0].ShipmentDate)
		sendMessageToClients(msg)
	} else {
		replyOrder.Order.Accepted = false
		replyOrder.Order.Reason = "OUT_OF_DATE"
	}
	json.NewEncoder(w).Encode(replyOrder)
}

// getOrderStatus отвечает на запрос с измененным статусом заказа
func getOrderStatus(w http.ResponseWriter, r *http.Request) {
	var inputOrder models.AcceptOrderRequest

	json.NewDecoder(r.Body).Decode(&inputOrder)

	switch inputOrder.Order.Substatus {
	case "STARTED":
		sendShipmentsInfo(inputOrder)
		sendStatus("PROCESSING", "READY_TO_SHIP", strconv.FormatInt(inputOrder.Order.ID, 10))
		msg := fmt.Sprintf("*Заказ №%d* передан в обработку - его можно начинать подготавливать\nПодробнее о заказе: /order%d", inputOrder.Order.ID, inputOrder.Order.ID)
		sendMessageToClients(msg)
	}
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
		resultMsg += fmt.Sprintf("*Заказ №%d*\n*Статус:* %s, *субстатус:* %s\nПодробнее о заказе: /order%d\n\n", order.ID, order.Status, order.Substatus, order.ID)
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
		msgText = fmt.Sprintf("*Заказ №%d:*\n", inputOrder.Order.ID)
		msgText += fmt.Sprintf("*Статус заказа:* %s, субстатус: %s\n", inputOrder.Order.Status, inputOrder.Order.Substatus)
		for i, item := range inputOrder.Order.Items {
			msgText += fmt.Sprintf("_Товар №%d:_\nOfferID товара: `%s`\nКоличество: `%d`\nЦена за шутку: `%.2f`\n", i+1, item.OfferID, item.Count, item.Price)
		}
		msgText += fmt.Sprintf("*Информация о доставке:*\nID посылки: `%d`\nДата отгрузки: `%s`\n", inputOrder.Order.Delivery.Shipments[0].ID, inputOrder.Order.Delivery.Shipments[0].ShipmentDate)
		msgText += fmt.Sprintf("*Ссылка на скачивание ярлыков-наклеек на грузовые места:*\n/label%d", inputOrder.Order.ID)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	msg.ParseMode = "markdown"
	if inputOrder.Order.Substatus == "STARTED" || inputOrder.Order.Substatus == "READY_TO_SHIP" {
		msg.ReplyMarkup = orderControlKeyboard
	}
	bot.Send(msg)
}

// sendStocksInfo отправляет Беру информацию об остатках товаров
func sendStocksInfo(w http.ResponseWriter, r *http.Request) {
	var stocksRequest models.StocksRequest
	var stocksResponse models.StocksResponse
	var count int64
	var updatedAt string

	json.NewDecoder(r.Body).Decode(&stocksRequest)

	err := db.Get(&updatedAt, "SELECT UPDATE_TIME FROM information_schema.tables WHERE TABLE_SCHEMA = 'xml2yml' AND TABLE_NAME = 'products'")
	if err != nil {
		log.WithFields(log.Fields{
			"function": "db.Get",
			"err":      err,
		},
		).Warn("Can't retrieve update time of products table")
	}
	updatedAt = strings.Replace(updatedAt, " ", "T", 1)
	updatedAt += "+03:00"
	for _, sku := range stocksRequest.Skus {
		var tempSku models.Skus
		var tempItem models.StocksItems
		err := db.Get(&count, "SELECT count FROM products WHERE shop_sku=?", sku)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "db.Get",
			},
			).Warn("Can't retrieve count of shop_sku, returning 0")
			count = 0
		}
		tempSku.Sku = sku
		tempSku.WarehouseID = stocksRequest.WarehouseID

		tempItem.UpdatedAt = updatedAt
		tempItem.Count = count
		tempItem.Type = "FIT"
		tempSku.Items = append(tempSku.Items, tempItem)
		stocksResponse.Skus = append(stocksResponse.Skus, tempSku)
	}
	json.NewEncoder(w).Encode(stocksResponse)
}