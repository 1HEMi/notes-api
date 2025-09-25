package update

import (
	"errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"net/http"
	JWTMiddleware "notes/internal/middleware"
	"notes/internal/storage"
	"notes/pkg/api/response"
	"notes/pkg/logger/sl"
	"strconv"
)

type Request struct {
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

type NoteUpdater interface {
	UpdateNote(noteID int, userID int, title, content string) error
}

func GetUserID(r *http.Request) (int, bool) {
	uid := JWTMiddleware.GetUserID(r.Context())
	if uid == 0 {
		return 0, false
	}
	return uid, true
}

func New(log *slog.Logger, noteUpdater NoteUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.update.New"
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
			log.Warn("user id mismatch", slog.Int("token_id", userIDFromToken), slog.Int("url_id", userIDFromURL))
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("forbidden access"))
			return
		}
		strNoteID := chi.URLParam(r, "note_id")
		noteID, err := strconv.Atoi(strNoteID)
		if err != nil {
			log.Error("invalid note id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid note id"))
			return
		}
		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.JSON(w, r, response.Error("failed to decode request body"))
			return
		}
		log.Info("decoded request", slog.Any("request", req))
		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			render.JSON(w, r, response.ValidationError(validateErr))
			return
		}
		err = noteUpdater.UpdateNote(noteID, userIDFromToken, req.Title, req.Content)
		if errors.Is(err, storage.ErrNoteNotFound) {
			log.Info("note not found", slog.Int("note_id", noteID))
			render.JSON(w, r, response.Error("note not found"))
			return
		}
		if errors.Is(err, storage.ErrForbidden) {
			log.Warn("forbidden update attempt",
				slog.Int("note_id", noteID),
				slog.Int("user_id", userIDFromToken),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("forbidden access"))
			return
		}
		if err != nil {
			log.Error("failed to update note", sl.Err(err))
			render.JSON(w, r, response.Error("failed to update note"))
			return
		}

		log.Info("note successfully updated", slog.Int("note_id", noteID))
		render.JSON(w, r, response.OK())

	}
}
