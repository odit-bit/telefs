package main

import (
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

	logger := logrus.New()

	worldNews, err := news.NewWNAPI(ctx, conf.ApiKey, conf.BackupDir,
		news.WithDebug(),
		news.WithLoadFromBackup(),
	)
	if err != nil {
		logger.Error(err, conf)
		return
	}

	bot, err := tApi.NewBotAPI(conf.TelegramToken)
	if err != nil {
		logger.Error(err)
		return
	}

	bot.Debug = true

	updt := tApi.NewUpdate(0)
	updt.Timeout = 60

	uChan := bot.GetUpdatesChan(updt)

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

	eg := errgroup.Group{}
	eg.Go(func() error {
		sig := <-sigC
		cancel()
		logger.Infof("got signal %v \n", sig)
		return nil
	})

updateLoop:
	for {
		select {
		case <-ctx.Done():
			break updateLoop
		case update, ok := <-uChan:
			if !ok {
				break updateLoop
			}

			if update.Message != nil {
				response := tApi.NewMessage(update.Message.Chat.ID, "")

				if update.Message.IsCommand() {
					switch update.Message.Command() {
					case "status":
						response.Text = fmt.Sprintf("online: %s", time.Now().UTC().String())
					case "topnews":
						top, _ := worldNews.TopNewsIndonesia()
						for _, l := range top {
							a := l.RandomArticle()
							response.Text = fmt.Sprintf("%s \n %s", a.Title, a.URL)
							_, err := bot.Send(response)
							if err != nil {
								logger.Error(err)
								continue
							}
						}
						continue
					default:
						response.Text = "sorry, i don't know the command"
					}
					if msg, err := bot.Send(response); err != nil {
						logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
					}
					continue
				}

			}
		}
	}

	if err := eg.Wait(); err != nil {
		logger.Errorf("error from errorgroup: %v", err)
	}
}
