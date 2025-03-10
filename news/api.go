package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/odit-bit/tdrive/news/wncli"
	"github.com/sirupsen/logrus"
)

const (
	_default_fetch_interval = 24 * time.Hour
	_backup_dir             = "./"
	_file_format_name       = "top_news"
)

type Service struct {
	*wnAPI
}

// this type will fetch and maintained news from World News API
type wnAPI struct {
	mx     sync.Mutex
	ctx    context.Context
	cache  map[string]wncli.TopNewsResp
	logger *logrus.Logger
	client *wncli.Client

	isRun     bool
	bootstrap bool

	fetchInterval time.Duration
	backupRoot    string
}

func NewWNAPI(ctx context.Context, apiKey string, backupDir string, opts ...option) (*Service, error) {
	// if err := conf.validate(); err != nil {
	// 	return nil, fmt.Errorf("invalid configuration: %v", err)
	// }
	if backupDir == "" {
		backupDir = _backup_dir
	}

	logger := logrus.New()

	cache := map[string]wncli.TopNewsResp{}
	wn := &wnAPI{
		mx:            sync.Mutex{},
		ctx:           ctx,
		cache:         cache,
		logger:        logger,
		fetchInterval: _default_fetch_interval,
		backupRoot:    backupDir,
		isRun:         false,
		client:        wncli.New(apiKey),
	}

	var err error
	for _, fn := range opts {
		wn, err = fn(wn)
		if err != nil {
			return nil, fmt.Errorf("wnApi error option: %v", err)
		}
	}

	// create backup dir if not exist
	_, err = os.Stat(wn.backupRoot)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(wn.backupRoot, 0o777); err != nil {
				return nil, fmt.Errorf("wnAPI failed create backup dir: %v", err)
			}
		} else {
			return nil, fmt.Errorf("wnAPI: %v", err)
		}
	}

	// if true, try load topnews from file
	if wn.bootstrap {
		if err := wn.load(); err != nil {
			err = fmt.Errorf("failed load from dump file: %v", err)
			wn.logger.Warn(err)
			wn.fetchTopNews(ctx)
			if err := wn.dump(); err != nil {
				wn.logger.Warnf("failed to dump data: %v", err)
			}
		}
	} else {
		wn.fetchTopNews(ctx)
	}

	go func() {

		now := time.Now()
		dur := wn.nextClockHour(now)
		timer := time.NewTicker(dur)
		defer timer.Stop()

		wn.logger.Debugf("next fetch time %s", now.Add(dur).String())
		for {

			select {
			case <-timer.C:
				wn.fetchTopNews(ctx)
				if err := wn.dump(); err != nil {
					logger.Warn("failed create dump: ", err)
				}
				timer.Reset(1 * time.Hour)
				wn.logger.Debugf("next fetch time %s", now.Add(1*time.Hour).String())

			case <-ctx.Done():
				return
			}
		}

	}()

	wn.isRun = true
	return &Service{wnAPI: wn}, nil
}

func (h *wnAPI) load() error {
	isLoad := false
	err := filepath.WalkDir(h.backupRoot, func(path string, d fs.DirEntry, err error) error {
		h.logger.Debugf("inspect backup file at dir:%v, name: %v", path, d.Name())
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.Contains(d.Name(), _file_format_name) {
			h.logger.Debugf("file name format is invalid: %v", d.Name())
			return nil
		}

		h.logger.Infof("load from file path:%s, name:%s", path, d.Name())
		name, _ := strings.CutSuffix(d.Name(), ".json")
		split := strings.Split(name, "_")

		if len(split) == 0 || split[len(split)-1] != "id" {
			return fmt.Errorf("invalid name format: %v", split)
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		tn := wncli.TopNewsResp{}
		if err := json.NewDecoder(f).Decode(&tn); err != nil {
			return err
		}

		if tn.CachedAt.Before(time.Now().Truncate(h.fetchInterval)) {
			return err
		}

		h.cache[(split[len(split)-1])] = tn
		isLoad = true
		return nil
	})

	if err != nil {
		return err
	}
	if !isLoad {
		return fmt.Errorf("backup file is not present")
	}
	return nil
}

func (h *wnAPI) fetchTopNews(ctx context.Context) {
	h.mx.Lock()
	defer h.mx.Unlock()

	result, err := h.client.TopNews(ctx, wncli.TN_ID)
	if err != nil {
		h.logger.Errorf("failed fetch topnews : %v", err)
		return
	}
	if len(result.TopNews) == 0 {
		h.logger.Errorf("failed fetch topnews : empty result, this is a bug \n dump: %v", result)
		return
	}
	result.CachedAt = time.Now()
	h.cache[string(wncli.TN_ID)] = *result
	h.logger.Infof("success fetch from world news api: %s", time.Now().String())
}

func (h *wnAPI) TopNewsIndonesia() ([]wncli.News, error) {
	h.mx.Lock()
	defer h.mx.Unlock()

	v := h.cache[string(wncli.TN_ID)]
	return v.TopNews, nil
}

func (h *wnAPI) dump() error {

	h.mx.Lock()
	defer h.mx.Unlock()

	for id, news := range h.cache {
		f, err := os.Create(filepath.Join(h.backupRoot, fmt.Sprintf("%s_%s.json", "top_news", string(id))))
		if err != nil {
			h.logger.Warnf("failed to save backup : %v \n", err)
			return err
		}
		if err := json.NewEncoder(f).Encode(news); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}

//////// helper

func (h *wnAPI) nextClockHour(t time.Time) time.Duration {
	future := t.Add(h.fetchInterval).Truncate(h.fetchInterval)

	dur := future.Sub(t)
	return dur
}

/////// options

var ErrOption = errors.New("wnApi is run cannot change/add options")

type option func(wn *wnAPI) (*wnAPI, error)

// try load from backup file , it will re-fetch if the loaded file is outdated
func WithLoadFromBackup() func(wn *wnAPI) (*wnAPI, error) {
	return func(wn *wnAPI) (*wnAPI, error) {
		if !wn.isRun {
			//Load from backup
			wn.bootstrap = true
			return wn, nil
		}
		return nil, ErrOption
	}
}

func WithDebug() option {
	return func(wn *wnAPI) (*wnAPI, error) {
		wn.logger.SetLevel(logrus.DebugLevel)
		return wn, nil
	}
}
