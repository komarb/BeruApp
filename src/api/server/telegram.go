package server

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	log "github.com/sirupsen/logrus"
	"reflect"
)

var bot *tgbotapi.BotAPI
var textShowOrders = "🛒 Показать открытые заказы 🛒"
var textDownloadAct = "📄 Скачать акт приема-передачи на сегодня 📄"
var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(textShowOrders),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton(textDownloadAct),
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
		if update.Message == nil {
			continue
		}
		if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {
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
					msg.ReplyMarkup = numericKeyboard
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
				msgText := "*Ссылка на акт:*\n" + fmt.Sprintf("https://api.partner.market.yandex.ru/v2/campaigns/%s/shipments/reception-transfer-act", cfg.Beru.CampaignID)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				msg.ParseMode = "markdown"
				bot.Send(msg)
			default:
				//msgText := getOpenOrder(getIdFromMsg(update.Message.Text))
				msgText := getOpenOrder()
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				msg.ParseMode = "markdown"
				bot.Send(msg)
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