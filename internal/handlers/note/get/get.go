package get

import (
	"errors"
	"log/slog"
	"net/http"
	"notes/internal/models"
	"notes/internal/storage"
	"notes/pkg/api/response"
	"notes/pkg/logger/sl"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type NoteGetter interface {
	GetNote(userID, noteID int) (*models.Note, error)
}

func New(log *slog.Logger, noteGetter NoteGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.get.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		StrUserID := chi.URLParam(r, "id")
		StrNoteID := chi.URLParam(r, "note_id")

		userID, err := strconv.Atoi(StrUserID)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid user id"))
			return
		}
		noteID, err := strconv.Atoi(StrNoteID)
		if err != nil {
			log.Error("invalid note id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid note id"))
			return
		}
		note, err := noteGetter.GetNote(userID, noteID)
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

		if note.UserID != userID {
			log.Warn("forbidden access to note",
				slog.Int("note_id", noteID),
				slog.Int("owner_id", note.UserID),
				slog.Int("requested_user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("forbidden access"))
			return
		}
		log.Info("note was delivered successfully", slog.Int("note_id", noteID))
		render.JSON(w, r, note)

	}
}
