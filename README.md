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
    "appPort": "8080"
  },
  "BotOptions": {
    "apiToken": "API token for Telegram bot"
  },
  "BeruOptions": {
    "campaignId" : "Campaign ID from profile"
  }
}
```
## Installation using Docker
Install Docker and in beruAPI/src/api directory execute command:
```
docker build -t beruapp . && docker run --name beruapp -d beruapp
```