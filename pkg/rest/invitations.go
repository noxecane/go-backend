package rest

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	ozzo "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/noxecane/anansi/api"
	sessions "github.com/noxecane/anansi/sessions"
	"tsaron.com/godview-starter/pkg/config"
	"tsaron.com/godview-starter/pkg/invitations"
	"tsaron.com/godview-starter/pkg/notification"
	"tsaron.com/godview-starter/pkg/users"
	"tsaron.com/godview-starter/pkg/workspaces"
)

var (
	isPhone        = regexp.MustCompile("0[789][01][0-9]{8,8}")
	errPhone       = ozzo.NewError("validation_is_phone", "must be a valid phone number(080xxxxxxxx)")
	phoneValidator = ozzo.NewStringRuleWithError(
		func(p string) bool {
			return isPhone.MatchString(p)
		},
		errPhone,
	)
)

type InvitationDTO struct {
	EmailAddress string `json:"email_address" mod:"smalltext"`
	Role         string `json:"role" mod:"smalltext"`
}

func (t *InvitationDTO) Validate() error {
	return ozzo.ValidateStruct(t,
		ozzo.Field(&t.EmailAddress, ozzo.Required, is.Email),
		ozzo.Field(&t.Role, ozzo.Required, ozzo.In("member", "admin")),
	)
}

type RegistrationDTO struct {
	CompanyName string `json:"company_name" mod:"trim"`
	FirstName   string `json:"first_name" mod:"trim"`
	LastName    string `json:"last_name" mod:"trim"`
	Password    string `json:"password" mod:"trim"`
	PhoneNumber string `json:"phone_number" mod:"trim"`
}

func (t *RegistrationDTO) Validate() error {
	return ozzo.ValidateStruct(t,
		ozzo.Field(&t.CompanyName),
		ozzo.Field(&t.FirstName, ozzo.Required),
		ozzo.Field(&t.LastName, ozzo.Required),
		ozzo.Field(&t.Password, ozzo.Required, ozzo.Length(8, 64)),
		ozzo.Field(&t.PhoneNumber, ozzo.Required, phoneValidator),
	)
}

func Invitations(r *chi.Mux, app *config.App, mailer notification.Mailer) {
	ivStore := invitations.NewStore(app.Tokens)
	uRepo := users.NewRepo(app.DB)
	wRepo := workspaces.NewRepo(app.DB)

	r.Route("/invitations", func(r chi.Router) {
		r.Post("/", inviteUsers(app.Auth, uRepo, ivStore, app.Env, mailer))
		r.Patch("/{token}/extend", extendInvitation(ivStore))
		r.Patch("/{token}/accept", acceptInvitation(ivStore, uRepo, wRepo, app.Auth))
	})
}

func extendInvitation(ivStore *invitations.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := api.StringParam(r, "token")

		iv, err := ivStore.Extend(r.Context(), token)
		if err != nil {
			if errors.Is(err, invitations.ErrExpired) {
				panic(api.Err{
					Code:    http.StatusUnauthorized,
					Message: "Your invitation token has expired",
				})
			}
			panic(err)
		}

		api.Success(r, w, iv)
	}
}

func acceptInvitation(ivStore *invitations.Store, uRepo *users.Repo, wRepo *workspaces.Repo, sStore *sessions.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dto RegistrationDTO
		api.ReadJSON(r, &dto)

		token := api.StringParam(r, "token")

		iv, err := ivStore.View(r.Context(), token)
		if err != nil {
			if errors.Is(err, invitations.ErrExpired) {
				panic(api.Err{
					Code:    http.StatusUnauthorized,
					Message: "Your invitation token has expired",
				})
			}
			panic(err)
		}

		user, err := uRepo.Register(r.Context(), iv.EmailAddress, users.Registration{
			FirstName:   dto.FirstName,
			LastName:    dto.LastName,
			PhoneNumber: dto.PhoneNumber,
			Password:    dto.Password,
		})
		if err != nil {
			if errors.Is(err, users.ErrExistingPhoneNumber) {
				panic(api.Err{
					Code:    http.StatusConflict,
					Message: err.Error(),
				})
			} else {
				panic(err)
			}
		}

		if err := ivStore.Revoke(r.Context(), user.EmailAddress); err != nil {
			panic(err)
		}

		workspace, err := wRepo.Get(r.Context(), user.Workspace)
		if err != nil {
			panic(api.Err{
				Code:    http.StatusForbidden,
				Message: "This workspace does not exist",
			})
		}

		session := session{
			Workspace:   user.Workspace,
			User:        user.ID,
			Role:        user.Role,
			CompanyName: workspace.CompanyName,
			FullName:    fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		}

		if err != nil {
			panic(err)
		}

		api.Success(r, w, session)
	}
}

func inviteUsers(auth *sessions.Store, uRepo *users.Repo, ivStore *invitations.Store, env *config.Env, mailer notification.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var session session
		api.Load(auth, r, &session)

		if session.Role == "member" {
			panic(api.Err{
				Code:    http.StatusForbidden,
				Message: "You are not allowed to invite other users",
			})
		}

		var dtos []InvitationDTO
		api.ReadJSON(r, &dtos)

		// create the invited users
		var reqs []users.UserRequest
		for _, dto := range dtos {
			reqs = append(reqs, users.UserRequest{
				EmailAddress: strings.ToLower(dto.EmailAddress),
				Role:         dto.Role,
			})
		}
		ux, err := uRepo.CreateMany(r.Context(), session.Workspace, reqs)
		if err != nil {
			panic(err)
		}

		// send them mail invitations
		var ivs []invitations.Invitation
		for _, u := range ux {
			iv, err := ivStore.Create(r.Context(), session.Workspace, session.CompanyName, u.EmailAddress)
			if err != nil {
				panic(err)
			}

			if err := invitations.SendInvitation(mailer, env.ClientUserPage, iv); err != nil {
				panic(err)
			}

			ivs = append(ivs, iv)
		}

		api.Success(r, w, ivs)
	}
}
