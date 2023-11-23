package workspaces

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

type Workspace struct {
	ID           uint      `bun:",pk" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	CompanyName  string    `json:"company_name"`
	EmailAddress string    `json:"email_address"`
}

type Repo struct {
	db bun.IDB
}

func NewRepo(db bun.IDB) *Repo {
	return &Repo{db}
}

// Create a workspace
func (r *Repo) Create(ctx context.Context, name, email string) (*Workspace, error) {
	workspace := &Workspace{CompanyName: name, EmailAddress: email}

	_, err := r.db.
		NewInsert().
		Model(workspace).
		Column("company_name", "email_address").
		Returning("*").
		Exec(ctx)

	return workspace, err
}

// Get returns the workspace with the given ID. Returns nil if the workspace doesn't exist
func (r *Repo) Get(ctx context.Context, id uint) (*Workspace, error) {
	workspace := &Workspace{ID: id}
	err := r.db.NewSelect().Model(workspace).WherePK().Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return workspace, err
}

// ChangeName updates the name of a workspace.
func (r *Repo) ChangeName(ctx context.Context, id uint, name string) (*Workspace, error) {
	workspace := &Workspace{
		ID:          id,
		CompanyName: name,
	}

	_, err := r.db.
		NewUpdate().
		Model(workspace).
		WherePK().
		Column("company_name").
		Returning("*").
		Exec(ctx)

	return workspace, err
}
