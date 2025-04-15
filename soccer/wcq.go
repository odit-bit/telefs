package soccer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	month = 30 * 24 * time.Hour
)

// underlying vendor is 3rd paty api endpoint that provide bounded context data for service
type VendorAPI interface {
	Events(ctx context.Context, from, to time.Time, param EventParam) ([]Fixture, error)
}

type Service struct {
	*wcq
}

type wcq struct {
	cached    []Fixture
	cli       VendorAPI
	backupDir string
}

func New(api VendorAPI, logger *logrus.Logger, backupDir string) (*Service, error) {
	logger.Info("INIT WCQ")
	w := &wcq{
		cached:    []Fixture{},
		cli:       api,
		backupDir: backupDir,
	}
	if err := w.load(context.Background()); err != nil {
		logger.Warn("wcq ", err, " try fetch from vendor")
		if err := w.fetchFixture(context.Background()); err != nil {
			logger.Info("wcq ", err)
			return nil, err
		}

		if err := w.dump(context.Background()); err != nil {
			logger.Error("wcq ", err)
			return nil, err
		}
		logger.Info("wcq fetch success")

	} else {
		logger.Info("wcq load from file success")
	}

	return &Service{w}, nil
}

func (wcq *wcq) fetchFixture(ctx context.Context) error {
	now := time.Now()

	fx, err := wcq.cli.Events(ctx, now, now.Add(1*month), EventParam{Timezone: "Asia/Jakarta", LeagueID: 22}) //(ctx, now, now.Add(1*month))
	if err != nil {
		return fmt.Errorf("wcq failed get upcoming fixture: %v", err)
	}

	wcq.cached = fx

	// sort.SliceStable(fx, func(i, j int) bool {
	// 	return fx[i].Date < fx[j].Date
	// })
	return nil
}

func (wcq *wcq) dump(_ context.Context) error {
	toFile := filepath.Join(wcq.backupDir, "Timnas_Fixtures.json")
	f, err := os.Create(toFile)
	if err != nil {
		return fmt.Errorf("wcq failed create backup file: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(wcq.cached); err != nil {
		return fmt.Errorf("wcq failed encode json: %v", err)
	}
	return nil
}

func (wcq *wcq) load(_ context.Context) error {
	toFile := filepath.Join(wcq.backupDir, "Timnas_Fixtures.json")
	f, err := os.Open(toFile)
	if err != nil {
		return fmt.Errorf("wcq failed open backup file")
	}
	defer f.Close()

	res := []Fixture{}
	if err := json.NewDecoder(f).Decode(&res); err != nil {
		return fmt.Errorf("wcq failed decode json: %v", err)
	}
	if len(res) == 0 {
		return fmt.Errorf("wcq loaded empty fixture from file")
	}

	wcq.cached = res
	return nil
}

func (wcq *wcq) Upcoming() ([]Fixture, bool) {
	if len(wcq.cached) == 0 {
		return nil, false
	}

	res := NearestFixtures(wcq.cached)
	return res, true
}
