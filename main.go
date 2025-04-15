package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/odit-bit/tdrive/news"
	"github.com/odit-bit/tdrive/soccer"
	"github.com/odit-bit/tdrive/soccer/afcom"
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

	// soccer service instance
	api := afcom.New(conf.TimnasEndpoint, conf.TimnasApiKey)
	timnasSvc, _ := soccer.New(api, logger, conf.TimnasBackupDir)

	// bot instance
	bot, err := tApi.NewBotAPI(conf.TelegramToken)
	if err != nil {
		logger.Error(err)
		return
	}

	// pubsub instance
	liveCL := soccer.NewLiveChampionLeague(api)
	defer liveCL.Close()

	bot.Debug = true

	// telegram bot update channel
	updt := tApi.NewUpdate(0)
	updt.Timeout = 120
	uChan := bot.GetUpdatesChan(updt)

	// listen os signal
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)
	eg := errgroup.Group{}

	eg.Go(func() error {
		sig := <-sigC
		cancel()
		liveCL.Close()
		logger.Infof("got signal %v \n", sig)
		return nil
	})

	eg.Go(func() error {
		return liveCL.PollContext(ctx)
	})

	var wg sync.WaitGroup
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

					case "wcq":
						timnasUpcomingFixCMD(timnasSvc, bot, &update, logger)

					case "subs":
						wg.Add(1)
						go func() {
							defer wg.Done()
							subscribeCMD(ctx, liveCL, bot, &update, logger)
						}()

					case "unsub":
						unsubCMD(liveCL, bot, &update)

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
	wg.Wait()
}

func unsubCMD(pub *soccer.ChampionsLeague, _ *tApi.BotAPI, update *tApi.Update) {
	resp := tApi.NewMessage(update.Message.Chat.ID, "")
	pub.RemoveConsumer(int(resp.ChatID))
}

func subscribeCMD(ctx context.Context, pub *soccer.ChampionsLeague, bot *tApi.BotAPI, update *tApi.Update, _ *logrus.Logger) {
	resp := tApi.NewMessage(update.Message.Chat.ID, "")
	sub := pub.AddConsumer(int(resp.ChatID))

	for {
		select {
		case ls, ok := <-sub.Event():
			if !ok {
				resp.Text = "channel closed"
				bot.Send(resp)
				return
			} else {
				resp.Text = fmt.Sprintf(
					"%s vs %s %s %s",
					ls.HomeTeam,
					ls.AwayTeam,
					ls.Score,
					ls.Minutes,
				)
			}
		case <-ctx.Done():
			return
		}
		bot.Send(resp)
	}
}

func timnasUpcomingFixCMD(wcq *soccer.Service, bot *tApi.BotAPI, update *tApi.Update, logger *logrus.Logger) {
	response := tApi.NewMessage(update.Message.Chat.ID, "")

	// get news
	res, ok := wcq.Upcoming()
	if !ok {
		// logger.Warnf("failed to fetch upcoming: %v", err)
		response.Text = "no upcoming fixture available, try again later."
		msg, err := bot.Send(response)
		if err != nil {
			logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
			return
		}
		return
	}

	if len(res) == 0 {
		logger.Error("timnas upcoming fixture is nil")
		return
	}

	// render news
	logger.Info("render upcoming fixture")
	var text bytes.Buffer
	for i, f := range res {
		text.Reset()
		text.WriteString(fmt.Sprintf("\n--%d. %s -- %s vs %s -- %s \r\n", i+1, f.StageName, f.HomeTeam, f.AwayTeam, f.Date))

		response.Entities = append(response.Entities, tApi.MessageEntity{
			Type: "bold",
		})
		response.Text = text.String()
		msg, err := bot.Send(response)
		if err != nil {
			logger.Errorf("failed to send message, messageID: %v, err: %v", msg.MessageID, err)
			return
		}
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
