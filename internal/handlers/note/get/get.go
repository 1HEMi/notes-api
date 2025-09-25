package get

import (
	"errors"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	JWTMiddleware "notes/internal/middleware"
	"notes/internal/models"
	"notes/internal/storage"
	"notes/pkg/api/response"
	"notes/pkg/logger/sl"
	"strconv"
)

type NoteGetter interface {
	GetNote(userID, noteID int) (*models.Note, error)
}

func GetUserID(r *http.Request) (int, bool) {
	uid := JWTMiddleware.GetUserID(r.Context())
	if uid == 0 {
		return 0, false
	}
	return uid, true
}
func New(log *slog.Logger, noteGetter NoteGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.get.New"

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
		note, err := noteGetter.GetNote(userIDFromToken, noteID)
		if errors.Is(err, storage.ErrNoteNotFound) {
			log.Info("note not found", slog.Int("note_id", noteID))
			render.JSON(w, r, response.Error("note not found"))
			return

		}
		if err != nil {
			log.Error("failed to get note", sl.Err(err))
			render.JSON(w, r, response.Error("failed to get note"))
			return
		}
		log.Info("note was delivered successfully", slog.Int("note_id", noteID))
		render.JSON(w, r, note)

	}
}
