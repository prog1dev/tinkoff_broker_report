## tinkoff_broker_report

Creates csv report of your trades via [Tinkoff Invest broker](www.tinkoff.ru/invest) within specified time frame. Can be uploaded to trades analyze serveces such as tradersync.com.

## Installation

```go get -u github.com/prog1dev/tinkoff_broker_report```

## Set up

First you need to generate token to be able to get your trades via tinkoff api. Last time it was down below [settings page](https://www.tinkoff.ru/invest/settings). Then set dates and result csv filepath and run ```go run main.go```
