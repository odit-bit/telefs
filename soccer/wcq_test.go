package soccer

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_wcq(t *testing.T) {

	////////
	toFile := "./Timnas_Fixtures.json"
	defer func() {
		os.Remove(toFile)
	}()

	expected := []Fixture{
		{ID: "1", CountryID: "22"},
		{ID: "2", CountryID: "23"},
	}

	w := wcq{cached: expected, backupDir: "./"}

	// DUMP
	if err := w.dump(context.Background()); err != nil {
		t.Fatal(err)
	}

	// mock Fetch
	f, err := os.OpenFile(toFile, os.O_RDWR, 0o666)
	if err != nil {
		t.Fatal(err)
	}

	expected[0].CountryName = "indonesia"
	expected[1].CountryName = "non-indonesia"

	if err := json.NewEncoder(f).Encode(expected); err != nil {
		t.Fatal(err)
	}
	f.Close()
	// mock fetch end

	// LOAD
	if err := w.load(context.Background()); err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, expected, w.cached)
}
