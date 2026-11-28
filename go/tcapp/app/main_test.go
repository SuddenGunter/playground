//go:build integration

package app_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/SuddenGunter/playground/go/tcapp/internal/testenv"
)

// pg is the Postgres container shared by all tests in this package.
var pg *testenv.Postgres

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	var err error

	pg, err = testenv.StartPostgres(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 1
	}

	defer func() {
		if err := pg.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	return m.Run()
}
