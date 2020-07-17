package server

import (
	"beruAPI/models"
	"context"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"time"
)

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

func createTables() {
	var queries []string
	queries = append(queries, `CREATE TABLE IF NOT EXISTS bot_clients (
	chatId VARCHAR(128) NOT NULL
		)`)
	/*queries = append(queries, `CREATE TABLE IF NOT EXISTS orders (
    currency VARCHAR(10),
    fake VARCHAR(5),
    id INT,
    paymentType VARCHAR(7),
    paymentMethod VARCHAR(6),
    taxSystem VARCHAR(3),
    delivery_region_id INT,
    delivery_shipments_id INT,
    delivery_shipments_shipmentDate DATETIME,
    delivery_serviceName VARCHAR(4),
    delivery_type VARCHAR(8),
    items_offerId INT,
    items_count INT,
    items_price INT,
    items_subsidy INT,
    items_vat VARCHAR(6)
);`)*/

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

func reserveItems(items []models.Items) error {
	count := -1
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, item := range items {
		err = db.Get(&count, "SELECT count FROM products WHERE offerId=?", item.OfferID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE products SET count=? WHERE offerId=?`, count-item.Count, item.OfferID)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}