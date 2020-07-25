package server

import (
	"context"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"time"
)

// initDB устанавливает соединение с БД и создает в ней необходимые таблицы
func initDB() {
	var err error
	log.Info("BeruAPI is starting up!")
	DSN := cfg.DB.Username + ":" + cfg.DB.Password + "@tcp(" + cfg.DB.Host + ":" + cfg.DB.DBPort + ")/" + cfg.DB.DBName
	log.WithField("dburi", DSN).Info("Current database DSN: ")

	db, err = sqlx.Connect("mysql", DSN)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "sql.Open",
			"error"	:	err},
		).Fatal("Failed to connect to MariaDB")
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = db.PingContext(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "sql.PingContext",
			"error"	:	err},
		).Fatal("Failed to ping MariaDB")
	}
	log.Info("Connected to MariaDB!")

	createTables()
}

// createTables создает необходимые для работы таблицы в БД
func createTables() {
	var queries []string
	queries = append(queries, `CREATE TABLE IF NOT EXISTS bot_clients (
	chatId VARCHAR(128) NOT NULL
		)`)
	queries = append(queries, `CREATE TABLE IF NOT EXISTS shipments (
	fulfilmentId VARCHAR(128) NOT NULL,
	offerId		VARCHAR(128) NOT NULL,
	count 		int
		)`)

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "createTables",
				"error"	:	err},
			).Fatal("Failed to create tables!")
		}
	}
}
func addShipmentToDB(fulfilmentId string, offerId string, count int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	tx.Exec(`INSERT INTO shipments (fulfilmentId, offerId, count) VALUES (?, ?, ?)`, fulfilmentId, offerId, count)
	err = tx.Commit()
	return nil
}

// subscribeChatId записывает ID пользователя в БД, если тот подписался на
// уведомления о новых заказах
func subscribeChatId(chatId int64) (bool, error) {
	selectedChatId := -1
	isSubscribed := false
	db.Get(&selectedChatId, `SELECT * FROM bot_clients WHERE chatId=?`, chatId)
	if selectedChatId != -1 {
		isSubscribed = true
	} else {
		tx, err := db.Begin()
		if err != nil {
			return isSubscribed, err
		}
		tx.Exec(`INSERT INTO bot_clients (chatId) VALUES (?)`, chatId)
		err = tx.Commit()
	}
	return isSubscribed, nil
}

// unsubscribeChatId удаляет ID пользователя из БД, если тот отписался от обновлений
func unsubscribeChatId(chatId int64) (bool, error) {
	selectedChatId := -1
	isSubscribed := true
	db.Get(&selectedChatId, `SELECT * FROM bot_clients WHERE chatId=?`, chatId)
	if selectedChatId != -1 {
		tx, err := db.Begin()
		if err != nil {
			return isSubscribed, err
		}
		tx.Exec(`DELETE FROM bot_clients WHERE chatId=?`, chatId)
		err = tx.Commit()
	} else {
		isSubscribed = false
	}
	return isSubscribed, nil
}