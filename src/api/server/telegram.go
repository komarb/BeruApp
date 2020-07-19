package server

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var bot *tgbotapi.BotAPI
var textShowOrders = "🛒 Показать открытые заказы 🛒"
var textDownloadAct = "📄 Скачать акт приема-передачи на сегодня 📄"
var menuKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(textShowOrders),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(textDownloadAct),
	),
)
var orderControlKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("❌ Отменить заказ ❌","confirmOrderCancellation"),
	),
)
var confirmKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Удалить заказ","doOrderCancellation"),
		tgbotapi.NewInlineKeyboardButtonData("Оставить заказ","undoOrderCancellation"),
	),
)
func runBot() {
	var err error
	bot, err = tgbotapi.NewBotAPI(cfg.Bot.ApiToken)
	if err != nil {
		log.Panic("Can't connect to bot, shutting down!")
	}
	log.WithFields(log.Fields{
		"botName": bot.Self.UserName,
	},
	).Info("Successfully connected to bot!")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			switch update.CallbackQuery.Data {
			case "confirmOrderCancellation":
				confirmOrderCancellation(update.CallbackQuery.Message)
			case "undoOrderCancellation":
				undoOrderCancellation(update.CallbackQuery.Message)
			case "doOrderCancellation":
				doOrderCancellation(update.CallbackQuery.Message)
			}
		} else if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {
			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! Я БеруБот, помогаю управлять заказами Беру. Чтобы подписаться на обновления, наберите команду '/subscribe'")
				bot.Send(msg)
			case "/subscribe":
				isSubscribed, err := subscribeChatId(update.Message.Chat.ID)
				if err != nil {
					log.WithFields(log.Fields{
						"function" : "subscribeChat",
						"error"	:	err},
					).Warn("Failed to subscribe user!")
				} else {
					msgText := ""
					if isSubscribed {
						msgText = "Вы уже подписаны на обновления!"
					} else {
						msgText = "Хорошие новости: теперь вы будете получать уведомления о новых заказах!"
					}
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					msg.ReplyMarkup = menuKeyboard
					bot.Send(msg)
				}
			case "/unsubscribe":
				isSubscribed, err := unsubscribeChatId(update.Message.Chat.ID)
				if err != nil {
					log.WithFields(log.Fields{
						"function" : "unsubscribeChat",
						"error"	:	err},
					).Warn("Failed to unsubscribe user!")
				} else {
					msgText := ""
					if isSubscribed {
						msgText = "Вы успешно отписались от обновлений!"
					} else {
						msgText = "Вы не подписаны на обновления!"
					}
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
					bot.Send(msg)
				}
			case textShowOrders:
				msgText := getOpenOrders()
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				msg.ParseMode = "markdown"
				bot.Send(msg)
			case textDownloadAct:
				downloadAct(update.Message.Chat.ID)
			case "/shipAllOrders":
				UpdateStatusToShippedAll()
			default:
				if strings.Contains(update.Message.Text, "/order") {
					getOpenOrder(getIdFromMsg(update.Message.Text), update.Message.Chat.ID)
				} else if strings.Contains(update.Message.Text, "/label") {
					downloadLabels(getIdFromMsg(update.Message.Text), update.Message.Chat.ID)
				} else if strings.Contains(update.Message.Text, "/shppd") {
					setShippedStatus(getIdFromMsg(update.Message.Text), update.Message.Chat.ID)
				} else {
					msgText := "Я вас не понимаю 😔 Отправьте команду /help для просмотра списка доступных команд"
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					msg.ParseMode = "markdown"
					bot.Send(msg)
				}
			}
		}
	}
}

func sendMessageToClients(msgText string) {
	var clientsID []int64
	err := db.Select(&clientsID, "SELECT * FROM bot_clients")
	if err != nil {
		log.WithFields(log.Fields{
			"function": "sendMessage",
			"error":    err},
		).Warn("Failed retrieve client list!")
	}
	for _, clientID := range clientsID {
		msg := tgbotapi.NewMessage(clientID, msgText)
		msg.ParseMode = "markdown"
		bot.Send(msg)
	}
}

func downloadAct(chatID int64) {
	actURL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/shipments/reception-transfer-act.json", cfg.Beru.CampaignID)
	resp := DoAuthRequest("GET", actURL, nil)
	if resp.StatusCode == 404 {
		msg := tgbotapi.NewMessage(chatID, "Заказы к отправке сегодняшним числом *отсутствуют*")
		msg.ParseMode = "markdown"
		bot.Send(msg)
	} else {
		var file tgbotapi.FileBytes
		file.Bytes, _ = ioutil.ReadAll(resp.Body)
		file.Name = fmt.Sprintf("act%s.pdf", time.Now().Format("02-01-2006"))
		msg := tgbotapi.NewDocumentUpload(chatID, file)
		bot.Send(msg)
	}
}

func downloadLabels(orderID string, chatID int64) {
	labelsURL := fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/orders/%s/delivery/labels.json", cfg.Beru.CampaignID, orderID)
	resp := DoAuthRequest("GET", labelsURL, nil)
	_, err := strconv.Atoi(orderID)
	if resp.StatusCode == 404 || err != nil{
		msg := tgbotapi.NewMessage(chatID, "Заказ с таким ID *не найден*")
		msg.ParseMode = "markdown"
		bot.Send(msg)
	} else {
		var file tgbotapi.FileBytes
		file.Bytes, _ = ioutil.ReadAll(resp.Body)
		file.Name = fmt.Sprintf("labels%s.pdf", orderID)
		msg := tgbotapi.NewDocumentUpload(chatID, file)
		bot.Send(msg)
	}
}

func confirmOrderCancellation(msg *tgbotapi.Message) {
	confirm := tgbotapi.NewEditMessageReplyMarkup(msg.Chat.ID, msg.MessageID, confirmKeyboard)
	bot.Send(confirm)
}

func undoOrderCancellation(msg *tgbotapi.Message) {
	orderControl := tgbotapi.NewEditMessageReplyMarkup(msg.Chat.ID, msg.MessageID, orderControlKeyboard)
	bot.Send(orderControl)
}

func doOrderCancellation(msg *tgbotapi.Message) {
	var statusMsgText string
	i := strings.Index(msg.Text, "/label")
	orderID := msg.Text[i+6:]
	resp := sendStatus("CANCELLED", "SHOP_FAILED", orderID)
	if resp.StatusCode != 200 {
		statusMsgText = fmt.Sprintf("Беру ответил ошибкой, заказ %s *не был отменен*!", orderID)
	} else {
		statusMsgText= fmt.Sprintf("Заказ %s успешно *отменен*!", orderID)
	}
	statusMsg := tgbotapi.NewMessage(msg.Chat.ID, statusMsgText)
	orderControl := tgbotapi.NewEditMessageReplyMarkup(msg.Chat.ID, msg.MessageID, orderControlKeyboard)
	bot.Send(orderControl)
	statusMsg.ParseMode = "markdown"
	bot.Send(statusMsg)
}

func setShippedStatus(orderID string, chatID int64) {
	var statusMsgText string
	resp := sendStatus("PROCESSING", "SHIPPED", orderID)
	if resp.StatusCode != 200 {
		statusMsgText = fmt.Sprintf("Беру ответил ошибкой, статус SHIPPED заказа %s *не был установлен*!", orderID)
	} else {
		statusMsgText= fmt.Sprintf("Статус SHIPPED заказа %s успешно *установлен*!", orderID)
	}
	statusMsg := tgbotapi.NewMessage(chatID, statusMsgText)
	statusMsg.ParseMode = "markdown"
	bot.Send(statusMsg)
}