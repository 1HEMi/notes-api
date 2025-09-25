package save

import (
	"errors"
	"log/slog"
	"net/http"
	"notes/internal/storage"
	"notes/pkg/api/response"
	"notes/pkg/auth"
	"notes/pkg/logger/sl"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserSaver interface {
	SaveUser(username, password string) (int, error)
}

func New(log *slog.Logger, userSaver UserSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.user.save.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
		var req Request
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode request"))
			return
		}
		log.Info("decoded request", slog.Any("request", req))
		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}

		userID, err := userSaver.SaveUser(req.Username, req.Password)
		if errors.Is(err, storage.ErrUserExists) {
			log.Info("username already exists", slog.String("username", req.Username))
			render.JSON(w, r, response.Error("username already exists"))
			return
		}
		if err != nil {
			log.Error("failed to create user", sl.Err(err))
			render.JSON(w, r, response.Error("failed to create user"))
			return
		}
		token, err := auth.GenerateToken(userID, req.Username)
		if err != nil {
			log.Error("failed to generate JWT", sl.Err(err))
			render.JSON(w, r, response.Error("failed to generate token"))
			return
		}
		log.Info("user successfully created", slog.String("username", req.Username))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]string{"token": token})
	}
}
