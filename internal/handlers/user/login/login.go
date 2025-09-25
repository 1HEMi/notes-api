package login

import (
	"errors"
	"log/slog"
	"net/http"
	"notes/internal/models"
	"notes/internal/storage"
	"notes/pkg/api/response"
	"notes/pkg/auth"
	"notes/pkg/logger/sl"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserSignIn interface {
	GetUserByUsername(username string) (*models.User, error)
}

func New(log *slog.Logger, userSignIn UserSignIn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.login.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("invalid request"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("validation failed", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		user, err := userSignIn.GetUserByUsername(req.Username)
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", slog.String("username", req.Username))
			render.JSON(w, r, response.Error("invalid username or password"))
			return
		}
		if err != nil {
			log.Error("failed to get user", sl.Err(err))
			render.JSON(w, r, response.Error("failed to get user"))
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			log.Warn("invalid password", slog.String("username", req.Username))
			render.JSON(w, r, response.Error("invalid username or password"))
			return
		}
		token, err := auth.GenerateToken(user.ID, user.Username)
		if err != nil {
			log.Error("failed to generate jwt token", sl.Err(err))
			render.JSON(w, r, response.Error("failed to generate token"))
			return
		}
		log.Info("user successfully logged in", slog.String("username", req.Username))

		render.JSON(w, r, map[string]string{
			"token": token,
		})
	}
}
