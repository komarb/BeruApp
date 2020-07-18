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
var httpClient *http.Client
func (a *App) Init(config *config.Config) {
	cfg = config
	httpClient = createHTTPClient()
	initDB()
	go runBot()
	go initPeriodicUpdate()
	a.Router = mux.NewRouter()
	a.setRouters()
}

func (a *App) setRouters() {
	public := a.Router.PathPrefix("/download").Subrouter()
	public.Use(loggingMiddleware)
	public.HandleFunc("/act", downloadActHand).Methods("GET")
	public.HandleFunc("/labels/{orderId}", downloadLabelsHand).Methods("GET")

	private := a.Router.PathPrefix("/api").Subrouter()
	private.Use(authMiddleware)
	private.HandleFunc("/stocks", sendStocksInfo).Methods("POST")
	private.HandleFunc("/cart", getItemsRelevantInfo).Methods("POST")
	private.HandleFunc("/order/accept", getOrderAcceptanceStatus).Methods("POST")
	private.HandleFunc("/order/status", getOrderStatus).Methods("POST")
}

func (a *App) Run(addr string) {
	var err error
	log.WithField("port", cfg.App.AppPort).Info("Starting server on port:")
	log.Info("Now handling routes!")

	if cfg.App.DaemonMode {
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

		log.WithFields(log.Fields{
			"PID"	:	d.Pid},
		).Info("Daemon started!")
	}
	if cfg.App.HttpsMode {
		err = http.ListenAndServeTLS(addr, cfg.App.CrtDir, cfg.App.KeyDir, a.Router)
	} else {
		err = http.ListenAndServe(addr, a.Router)
	}
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