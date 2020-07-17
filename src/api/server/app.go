package server

import (
	"beruAPI/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/sevlyar/go-daemon"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
)

type App struct {
	Router *mux.Router
	DB *mongo.Client
}

var cfg *config.Config
var db *sqlx.DB
func (a *App) Init(config *config.Config) {
	cfg = config
	initDB()
	go runBot()
	a.Router = mux.NewRouter()
	a.setRouters()
}

func (a *App) setRouters() {
	a.Router.Use(loggingMiddleware)

	a.Router.HandleFunc("/cart", getItemsRelevantInfo).Methods("POST")
	a.Router.HandleFunc("/order/accept", getOrderAcceptanceStatus).Methods("POST")
	a.Router.HandleFunc("/order/status", changeOrderStatus).Methods("POST")
}

func (a *App) Run(addr string) {
	log.WithField("port", cfg.App.AppPort).Info("Starting server on port:")
	log.Info("Now handling routes!")

	cntxt := &daemon.Context{
		PidFileName: "beruapp.pid",
		PidFilePerm: 0644,
		LogFileName: "beruapp.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	if d != nil {
		return
	}
	defer cntxt.Release()

	log.Info("Daemon started!")

	err = http.ListenAndServe(addr, a.Router)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "http.ListenAndServe",
			"error"	:	err},
		).Fatal("Failed to run a server!")
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}