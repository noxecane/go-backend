package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleMember = "member"
	RoleAdmin  = "admin"
	RoleOwner  = "owner"
)

var ErrExistingPhoneNumber = errors.New("tphone number already in use")
var ErrExistingEmail = errors.New("email already in use")

type Registration struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Password    string `json:"password"`
	PhoneNumber string `json:"phone_number"`
}

type User struct {
	ID           uint      `bun:",pk" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	FirstName    string    `json:"first_name,omitempty"`
	LastName     string    `json:"last_name,omitempty"`
	Role         string    `json:"role"`
	Password     []byte    `json:"-"`
	EmailAddress string    `json:"email_address"`
	PhoneNumber  string    `json:"phone_number,omitempty"`
	Workspace    uint      `json:"workspace"`
}

type UserRequest struct {
	EmailAddress string
	Role         string
}

type Repo struct {
	db bun.IDB
}

func NewRepo(db bun.IDB) *Repo {
	return &Repo{db}
}

func (r *Repo) Create(ctx context.Context, workspace uint, req UserRequest) (*User, error) {
	user := &User{
		EmailAddress: req.EmailAddress,
		Role:         req.Role,
		Workspace:    workspace,
	}

	_, err := r.db.
		NewInsert().
		Model(user).
		Column("email_address", "role", "workspace").
		Returning("*").
		Exec(ctx)

	if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
		return nil, ErrExistingEmail
	}

	return user, err
}

func (r *Repo) CreateMany(ctx context.Context, workspace uint, reqs []UserRequest) ([]User, error) {
	var users []User

	for _, req := range reqs {
		users = append(users, User{
			EmailAddress: req.EmailAddress,
			Role:         req.Role,
			Workspace:    workspace,
		})
	}

	_, err := r.db.
		NewInsert().
		Model(&users).
		Column("email_address", "role", "workspace").
		Returning("*").
		Exec(ctx)

	if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
		return nil, ErrExistingEmail
	}

	return users, err
}

func (r *Repo) Register(ctx context.Context, email string, reg Registration) (*User, error) {
	pwdBytes, err := bcrypt.GenerateFromPassword([]byte(reg.Password), 10)
	if err != nil {
		return nil, err
	}

	user := &User{
		Password:    pwdBytes,
		FirstName:   reg.FirstName,
		LastName:    reg.LastName,
		PhoneNumber: reg.PhoneNumber,
	}

	_, err = r.db.
		NewUpdate().
		Model(user).
		Where("email_address = ?", email).
		Column("first_name", "last_name", "phone_number", "password").
		Returning("*").
		Exec(ctx)

	if err, ok := err.(*pgconn.PgError); ok && err.Code == pgerrcode.UniqueViolation {
		return nil, ErrExistingPhoneNumber
	}

	return user, err
}

func (r *Repo) ChangePassword(ctx context.Context, wkID, id uint, password string) (*User, error) {
	pwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return nil, err
	}

	user := &User{Password: pwdBytes}
	_, err = r.db.
		NewUpdate().
		Model(user).
		Where("id = ?", id).
		Where("workspace = ?", wkID).
		Column("password").
		Returning("*").
		Exec(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return user, err
}
