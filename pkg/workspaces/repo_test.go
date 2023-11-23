package workspaces

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/noxecane/anansi"
	"github.com/uptrace/bun"
	"syreclabs.com/go/faker"
	"tsaron.com/godview-starter/pkg/config"
)

var testDB *bun.DB

func afterEach(t *testing.T) {
	if _, err := testDB.NewTruncateTable().Table("workspaces", "users").Cascade().Exec(context.TODO()); err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	var err error
	var sqlDB *sql.DB

	var env config.Env
	if err = anansi.LoadEnv(&env); err != nil {
		panic(err)
	}

	log := anansi.NewLogger(env.Name)

	if sqlDB, testDB, err = config.SetupDB(env); err != nil {
		panic(err)
	}
	log.Info().Msg("Successfully connected to postgres")

	code := m.Run()

	if err := sqlDB.Close(); err != nil {
		log.Err(err).Msg("Failed to disconnect from postgres cleanly")
	}

	os.Exit(code)
}

func TestRepoGetByID(t *testing.T) {
	repo := NewRepo(testDB)
	ctx := context.TODO()

	t.Run("returns a workspace based on its ID", func(t *testing.T) {
		defer afterEach(t)

		wk, err := repo.Create(ctx, faker.Company().Name(), faker.Internet().Email())
		if err != nil {
			t.Fatal(err)
		}

		wk2, err := repo.Get(ctx, wk.ID)
		if err != nil {
			t.Fatal(err)
		}

		if wk2.ID != wk.ID {
			t.Errorf("Expected loaded workspace(%d) to be the same as created workspace(%d)", wk2.ID, wk.ID)
		}
	})

	t.Run("return nil if the workspace doesn't exist", func(t *testing.T) {
		defer afterEach(t)

		wk, err := repo.Get(ctx, uint(faker.RandomInt(1, 20)))
		if err != nil {
			t.Fatal(err)
		}

		if wk != nil {
			t.Errorf("Expected loaded workspace to be the nil found %v", *wk)
		}
	})
}
