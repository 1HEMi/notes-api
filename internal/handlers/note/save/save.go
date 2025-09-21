package save

import (
	"log/slog"
	"net/http"

	"notes/pkg/api/response"
	"notes/pkg/logger/sl"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

type NoteSaver interface {
	SaveNote(userID int, title, content string) error
}

func New(log *slog.Logger, noteSaver NoteSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.save.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
		userID := chi.URLParam(r, "id")
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

		uid, err := strconv.Atoi(userID)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid user id"))
			return
		}

		err = noteSaver.SaveNote(uid, req.Title, req.Content)
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
