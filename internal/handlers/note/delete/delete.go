package delete

import (
	"errors"
	"log/slog"
	"net/http"
	"notes/internal/storage"
	"notes/pkg/api/response"
	"notes/pkg/logger/sl"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type NoteDeleter interface {
	DeleteNote(noteID, userID int) error
}

func New(log *slog.Logger, noteDeleter NoteDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		strUserID := chi.URLParam(r, "id")
		userID, err := strconv.Atoi(strUserID)
		if err != nil {
			log.Error("invalid user id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid user id"))
			return
		}

		strNoteID := chi.URLParam(r, "note_id")
		noteID, err := strconv.Atoi(strNoteID)
		if err != nil {
			log.Error("invalid note id", sl.Err(err))
			render.JSON(w, r, response.Error("invalid note id"))
			return
		}
		err = noteDeleter.DeleteNote(noteID, userID)
		if errors.Is(err, storage.ErrNoteNotFound) {
			log.Info("note not found", slog.Int("note_id", noteID))
			render.JSON(w, r, response.Error("note not found"))
			return
		}
		if errors.Is(err, storage.ErrForbidden) {
			log.Warn("forbidden delete attempt",
				slog.Int("note_id", noteID),
				slog.Int("user_id", userID),
			)
			render.Status(r, http.StatusForbidden)
			render.JSON(w, r, response.Error("forbidden access"))
			return
		}
		if err != nil {
			log.Error("failed to delete note", sl.Err(err))
			render.JSON(w, r, response.Error("failed to delete note"))
			return
		}

		log.Info("note successfully deleted", slog.Int("note_id", noteID))
		render.JSON(w, r, response.OK())
	}
}
