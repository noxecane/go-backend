package users

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"noxecane/go-starter/pkg/config"

	"noxecane/go-starter/pkg/workspaces"

	"github.com/jaswdr/faker"
	"github.com/noxecane/anansi"
	"github.com/uptrace/bun"
)

var testDB *bun.DB
var fake = faker.New()

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

func TestRepoCreate(t *testing.T) {
	defer afterEach(t)

	repo := NewRepo(testDB)
	ctx := context.TODO()

	wkRepo := workspaces.NewRepo(testDB)
	wk, err := wkRepo.Create(ctx, fake.Company().Name(), fake.Internet().Email())
	if err != nil {
		t.Fatal(err)
	}

	req := UserRequest{fake.Internet().Email(), fake.Company().JobTitle()}
	_, err = repo.Create(ctx, wk.ID, req)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.Create(ctx, wk.ID, req)
	if err == nil {
		t.Fatalf("Expected duplicate create to fail")
	}

	if err != ErrExistingEmail {
		t.Errorf("Expected error to be of type errEmail, got %T", err)
	}
}

func TestRepoCreateMany(t *testing.T) {
	defer afterEach(t)

	repo := NewRepo(testDB)
	ctx := context.TODO()

	wkRepo := workspaces.NewRepo(testDB)
	wk, err := wkRepo.Create(ctx, fake.Company().Name(), fake.Internet().Email())
	if err != nil {
		t.Fatal(err)
	}

	req := UserRequest{fake.Internet().Email(), fake.Company().JobTitle()}
	_, err = repo.Create(ctx, wk.ID, req)
	if err != nil {
		t.Fatal(err)
	}

	reqs := []UserRequest{
		{fake.Internet().Email(), fake.Company().JobTitle()},
		req,
	}
	_, err = repo.CreateMany(ctx, wk.ID, reqs)
	if err == nil {
		t.Fatalf("Expected duplicate create to fail")
	}

	if err != ErrExistingEmail {
		t.Errorf("Expected error to be of type ErrEmail, got %T: %v", err, err)
	}
}

func TestRepoRegister(t *testing.T) {
	defer afterEach(t)

	repo := NewRepo(testDB)
	ctx := context.TODO()

	wkRepo := workspaces.NewRepo(testDB)
	wk, err := wkRepo.Create(ctx, fake.Company().Name(), fake.Internet().Email())
	if err != nil {
		t.Fatal(err)
	}

	reqs := []UserRequest{
		{fake.Internet().Email(), fake.Company().JobTitle()},
		{fake.Internet().Email(), fake.Company().JobTitle()},
	}
	_, err = repo.CreateMany(ctx, wk.ID, reqs)
	if err != nil {
		t.Fatal(err)
	}

	reg := Registration{
		fake.Person().FirstName(),
		fake.Person().LastName(),
		fake.Lorem().Word(),
		fake.Phone().Number(),
	}

	_, err = repo.Register(ctx, reqs[0].EmailAddress, reg)
	if err != nil {
		t.Fatal(err)
	}

	reg2 := Registration{
		fake.Person().FirstName(),
		fake.Person().LastName(),
		fake.Lorem().Word(),
		reg.PhoneNumber,
	}
	_, err = repo.Register(ctx, reqs[1].EmailAddress, reg2)
	if err != ErrExistingPhoneNumber {
		t.Errorf("Expected registeration with \"%v\", got %v", ErrExistingPhoneNumber, err)
	}
}
