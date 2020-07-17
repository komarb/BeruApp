package config

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
)

type Config struct {
	DB   *DBConfig     `json:"DbOptions"`
	App  *AppConfig    `json:"AppOptions"`
	Bot  *BotConfig	   `json:"BotOptions"`
	Beru *BeruConfig   `json:"BeruOptions"`
}

type DBConfig struct {
	Username	string		`json:"username"`
	Password	string		`json:"password"`
	Host 		string		`json:"host"`
	DBPort 		string		`json:"dbPort"`
	DBName 		string		`json:"dbName"`
}
type AppConfig struct {
	AppPort		string	`json:"appPort"`
	DaemonMode	bool 	`json:"daemonMode"`
	HttpsMode	bool	`json:"httpsMode"`
	CrtDir		string	`json:"crtDir"`
	KeyDir		string	`json:"keyDir"`
}
type BotConfig struct {
	ApiToken	string	`json:"apiToken"`
}
type BeruConfig struct {
	ApiToken	string	`json:"apiToken"`
	CampaignID	string	`json:"campaignId"`
	OAuthToken  string	`json:"oauthToken"`
	OAuthClientID  string	`json:"oauthClientId"`
}

func GetConfig() *Config {
	var config Config
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "GetConfig.ReadFile",
			"error"	:	err,
		},
		).Fatal("Can't read config.json file, shutting down...")
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "GetConfig.Unmarshal",
			"error"	:	err,
		},
		).Fatal("Can't correctly parse json from config.json, shutting down...")
	}
	return &config
}
