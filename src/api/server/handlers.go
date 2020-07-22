package server

import (
	"beruAPI/models"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		msgText := getOrderInfo(strconv.FormatInt(inputOrder.Order.ID,10))
		sendMessageToClients(msgText)
	}
	w.WriteHeader(200)
}

// sendStocksInfo отправляет Беру информацию об остатках товаров
func sendStocksInfo(w http.ResponseWriter, r *http.Request) {
	var stocksRequest models.StocksRequest
	var stocksResponse models.StocksResponse

	json.NewDecoder(r.Body).Decode(&stocksRequest)

	for _, sku := range stocksRequest.Skus {
		var tempSku models.Skus
		var tempItem models.StocksItems
		err := db.Get(&tempItem, "SELECT count, updated_at FROM products WHERE shop_sku=?", sku)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "db.Get",
			},
			).Warn("Can't retrieve count of shop_sku, returning 0")
			tempItem.Count = 0
			tempItem.UpdatedAt = time.Now().Local().Format(time.RFC3339)
			tempItem.UpdatedAt = strings.Replace(tempItem.UpdatedAt, "Z", "+", -1)
		}
		tempSku.Sku = sku
		tempSku.WarehouseID = stocksRequest.WarehouseID

		tempItem.Type = "FIT"
		tempSku.Items = append(tempSku.Items, tempItem)
		stocksResponse.Skus = append(stocksResponse.Skus, tempSku)
	}
	json.NewEncoder(w).Encode(stocksResponse)
}