package server

import (
	"beruAPI/models"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

func getItemsRelevantInfo(w http.ResponseWriter, r *http.Request) {
	var inputCart models.RelevantCartRequest
	var relevantItem models.Item
	var relevantCart models.RelevantCartRequest
	count := 0
	notInStock := 0
	json.NewDecoder(r.Body).Decode(&inputCart)
	for _, item := range inputCart.Cart.Items {
		err := db.Get(&count, "SELECT count FROM products WHERE offerId=?", item.OfferId)
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

func getOrderAcceptanceStatus(w http.ResponseWriter, r *http.Request) {
	var inputOrder models.AcceptOrderRequest
	var replyOrder models.ReplyOrderRequest
	var acceptOrder bool
	count := 0
	json.NewDecoder(r.Body).Decode(&inputOrder)
	for _, item := range inputOrder.Order.Items {
		log.Info(item.OfferID)
		err := db.Get(&count, "SELECT count FROM products WHERE offerId=?", item.OfferID)
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
		err := reserveItems(inputOrder.Order.Items)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "reserveItems",
				"handler" : "getOrderAcceptanceStatus",
				"err" : err,
			},
			).Fatal("Can't reserve item, declining an order")
		}
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

func changeOrderStatus(w http.ResponseWriter, r *http.Request) {
	var inputOrder models.AcceptOrderRequest

	json.NewDecoder(r.Body).Decode(&inputOrder)

	switch inputOrder.Order.Substatus {
	case "STARTED":
		sendShipmentsInfo(inputOrder)
		msg := fmt.Sprintf("*Заказ №%d* передан в обработку - его можно начинать подготавливать\nПодробнее о заказе: /order%d", inputOrder.Order.ID, inputOrder.Order.ID)
		sendMessageToClients(msg)
	}
}

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
	fmt.Printf("%#v", shipment)
	/*URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%d/delivery/shipments/%d/boxes.json",
		cfg.Beru.CampaignID, inputOrder.Order.ID, inputOrder.Order.Delivery.Shipments[0].ID)
	req, err := http.NewRequest("PUT", URL, nil)*/
}

func getOpenOrders() string {
	var inputOrders models.OpenOrdersRequest
	var resultMsg string
	//URL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders.json", cfg.Beru.CampaignID)
	URL := "https://pastebin.com/raw/h0ZSNHr1"
	req, err := http.Get(URL)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "getOpenOrders",
		},
		).Warn("Can't retrieve open orders!")
	}

	json.NewDecoder(req.Body).Decode(&inputOrders)
	resultMsg += fmt.Sprintf("Всего заказов: %d\n\n", len(inputOrders.Orders))
	for _, order := range inputOrders.Orders {
		resultMsg += fmt.Sprintf("*Заказ №%d*\nПодробнее о заказе: /order%d\n\n", order.ID, order.ID)
	}
	return resultMsg
}

func getOpenOrder() string {
	var inputOrder models.AcceptOrderRequest
	var resultMsg string
	//orderURL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders.json", cfg.Beru.CampaignID)
	orderURL := "https://pastebin.com/raw/48FvKiq9"
	req, err := http.Get(orderURL)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "getOpenOrder",
		},
		).Warn("Can't retrieve order by ID!")
	}
	json.NewDecoder(req.Body).Decode(&inputOrder)

	labelsURL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%d/delivery/labels.json", cfg.Beru.CampaignID, inputOrder.Order.ID)
	resultMsg = fmt.Sprintf("*Заказ №%d:*\n", inputOrder.Order.ID)
	//resultMsg += fmt.Sprintf("*Статус заказа: %s, субстатус: %s\n", inputOrder.Order.Status, inputOrder.Order.Substatus)
	for i, item := range inputOrder.Order.Items {
		resultMsg += fmt.Sprintf("_Товар №%d:_\nOfferID товара: `%s`\nКоличество: `%d`\nЦена за шутку: `%.2f`\n", i+1, item.OfferID, item.Count, item.Price)
	}
	resultMsg += fmt.Sprintf("*Информация о доставке:*\nID посылки: `%d`\nДата отгрузки: `%s`\n", inputOrder.Order.Delivery.Shipments[0].ID, inputOrder.Order.Delivery.Shipments[0].ShipmentDate)
	resultMsg += fmt.Sprintf("*Ссылка на скачивание ярлыков-наклеек на грузовые места:*\n%s", labelsURL)
	return resultMsg
}