package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/odit-bit/tdrive/news"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func main() {
	var configFile string

	flag.StringVar(&configFile, "config", "", "path to config file default './config.yaml'")
	flag.Parse()

	conf, err := LoadConfig(configFile)
	if err != nil {
		log.Fatal("failed load config: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// logger instance
	logger := logrus.New()

	// news service instance
	worldNews, err := news.NewWNAPI(ctx, conf.ApiKey, conf.BackupDir,
		news.WithDebug(),
		news.WithLoadFromBackup(),
	)
	if err != nil {
		logger.Error(err, conf)
		return
	}

	// bot instance
	bot, err := tApi.NewBotAPI(conf.TelegramToken)
	if err != nil {
		logger.Error(err)
		return
	}

	bot.Debug = true

	// update channel
	updt := tApi.NewUpdate(0)
	updt.Timeout = 60
	uChan := bot.GetUpdatesChan(updt)

	// listen os signal
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
	eg := errgroup.Group{}
	eg.Go(func() error {
		sig := <-sigC
		cancel()
		logger.Infof("got signal %v \n", sig)
		return nil
	})

listenUpdate:
	for {
		select {

		case <-ctx.Done():
			break listenUpdate

		case update, ok := <-uChan:
			if !ok {
				break listenUpdate
			}

			if update.Message != nil {
				if update.Message.IsCommand() {
					switch update.Message.Command() {
					case "status":
						statusCMD(bot, &update, logger)

					case "top":
						topnewsCmd(bot, &update, worldNews, logger)

					default:
						unknownCMD(bot, &update, logger)
					}
				}
			}

		}
	}

	if err := eg.Wait(); err != nil {
		logger.Errorf("error from errorgroup: %v", err)
	}
}

func unknownCMD(bot *tApi.BotAPI, update *tApi.Update, logger *logrus.Logger) {
	response := tApi.NewMessage(update.Message.Chat.ID, "")
	response.Text = "sorry, i don't know the command"
	if msg, err := bot.Send(response); err != nil {
		logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
	}
}

func statusCMD(bot *tApi.BotAPI, update *tApi.Update, logger *logrus.Logger) {
	response := tApi.NewMessage(update.Message.Chat.ID, "")
	response.Text = fmt.Sprintf("online: %s", time.Now().UTC().String())
	if msg, err := bot.Send(response); err != nil {
		logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
	}
}

func topnewsCmd(bot *tApi.BotAPI, update *tApi.Update, worldNews *news.Service, logger *logrus.Logger) {
	response := tApi.NewMessage(update.Message.Chat.ID, "")

	// get news
	top, err := worldNews.TopNewsIndonesia()
	if err != nil {
		logger.Warnf("failed to fetch top news: %v", err)
		response.Text = "failed to fetch top news, try again later."
		msg, err := bot.Send(response)
		if err != nil {
			logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
			return
		}
		return
	}

	// render news
	var text bytes.Buffer
	for _, l := range top {
		text.Reset()
		for i, a := range l.Articles {
			text.WriteString(fmt.Sprintf("\n--%d \n %s \n %s \r\n", i, a.Title, a.URL))
		}

		response.Text = text.String()
		msg, err := bot.Send(response)
		if err != nil {
			logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
			return
		}
	}
}
