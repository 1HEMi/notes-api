package save

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	JWTMiddleware "notes/internal/middleware"
	"notes/pkg/api/response"
	"notes/pkg/logger/sl"
	"strconv"
)

type Request struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

type NoteSaver interface {
	SaveNote(userID int, title, content string) error
}

func GetUserID(r *http.Request) (int, bool) {
	uid := JWTMiddleware.GetUserID(r.Context())
	if uid == 0 {
		return 0, false
	}
	return uid, true
}

func New(log *slog.Logger, noteSaver NoteSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.save.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
		userIDFromToken, ok := GetUserID(r)
		if !ok {
			log.Error("unauthorized: no user_id in context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, response.Error("unauthorized"))
			return
		}

		strUserID := chi.URLParam(r, "id")
		userIDFromURL, err := strconv.Atoi(strUserID)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid user id"))
			return
		}
		if userIDFromToken != userIDFromURL {
			log.Warn("user id mismatch",
				slog.Int("token_id", userIDFromToken),
				slog.Int("url_id", userIDFromURL),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("forbidden"))
			return
		}
		var req Request
		err = render.DecodeJSON(r.Body, &req)
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

		err = noteSaver.SaveNote(userIDFromToken, req.Title, req.Content)
		if err != nil {
			log.Error("failed to create note", sl.Err(err))
			render.JSON(w, r, response.Error("failed to create note"))
			return
		}
		log.Info("note successfully created", slog.String("title", req.Title))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, response.OK())
	}
}
