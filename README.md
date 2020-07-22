# BeruApp
Application for managing orders from beru.ru

## Configuration

File ```src/ITLabReports/api/config.json``` must contain next content:

```js
{
  "DbOptions": {
    "username" : "DB username",
    "password" : "DB pass",
    "host": "DB host",
    "dbPort": "DB port",
    "dbName" : "DB table name"
  },
  "AppOptions": {
    "appPort": "8080",
    "daemonMode": false | true,
    "httpsMode": false | true,
    "testMode" : false | true,
    "crtDir" : "path to certificate",
    "keyDir" : "path to key"
  },
  "BotOptions": {
    "apiToken": "API token for Telegram bot"
  },
  "BeruOptions": {
    "oauthToken" : "Yandex App OAuth Token",
    "oauthClientId": "Yandex App client ID",
    "apiToken": "Beru API token",
    "campaignId" : "Beru Campaign ID from profile"
  }
}
```
## Installation using Docker
Install Docker and in beruAPI/src/api directory execute command:
```
docker build -t beruapp . && docker run --name beruapp -d beruapp
```