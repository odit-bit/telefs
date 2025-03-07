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
	// var backupDIR string

	// flag.StringVar(&backupDIR, "backup-dir", "", "backup file directory")
	flag.StringVar(&configFile, "config", "", "path to config file default './config.yaml'")
	// flag.StringVar(&configF)
	flag.Parse()

	conf, err := LoadConfig(configFile)
	if err != nil {
		log.Fatal("failed load config: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

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
					bot.Send(response)
					continue
				}

				// if update.Message.Document != nil {
				// 	err := StoreDoc(ctx, bot, &update)
				// 	if err != nil {
				// 		logger.Errorf("failed get file from bot api: %v", err)
				// 		response.Text = "failed store document"
				// 	} else {
				// 		response.Text = "success"
				// 	}
				// 	bot.Send(response)
				// }
			}
		}
	}

	eg.Wait()
}

// func StoreDoc(ctx context.Context, bot *tApi.BotAPI, updt *tApi.Update) error {

// 	file, err := bot.GetFile(tApi.FileConfig{FileID: updt.Message.Document.FileID})
// 	if err != nil {
// 		return err
// 	}
// 	link := file.Link(token)
// 	resp, err := http.Get(link)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != 200 && resp.StatusCode != 202 {
// 		return fmt.Errorf("error status code %v", resp.Status)
// 	}

// 	id := strconv.FormatInt(updt.Message.From.ID, 10)
// 	toDir := filepath.Join("/mnt/d/wsl/tdrive", id)
// 	os.MkdirAll(toDir, 0o777)
// 	toFile := filepath.Join(toDir, updt.Message.Document.FileName)
// 	f, err := os.Create(toFile)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()

// 	if _, err := io.Copy(f, resp.Body); err != nil {
// 		return err
// 	}
// 	return nil
// }
