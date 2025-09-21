package getall

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

type AllNoteGetter interface {
	GetAllNotes(userID, limit, offset int, sort string) ([]models.Note, error)
}

func New(log *slog.Logger, allNoteGetter AllNoteGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.getall.New"

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

		limit := 3
		offset := 0
		sort := "desc"

		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				limit = v
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil && v > 0 {
				offset = v
			}
		}
		if s := r.URL.Query().Get("sort"); s == "asc" {
			sort = "asc"
		}

		notes, err := allNoteGetter.GetAllNotes(userID, limit, offset, sort)
		if errors.Is(err, storage.ErrNoteNotFound) {
			log.Info("notes not found")
			render.JSON(w, r, response.Error("notes not found"))
			return

		}
		if err != nil {
			log.Error("failed to get notes", sl.Err(err))
			render.JSON(w, r, response.Error("failed to get notes"))
			return
		}

		for _, note := range notes {
			if note.UserID != userID {
				log.Warn("forbidden access to notes",
					slog.Int("owner_id", note.UserID),
					slog.Int("requested_user_id", userID),
				)
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, response.Error("forbidden access"))
				return
			}
		}
		log.Info("notes was delivered successfully")
		render.JSON(w, r, notes)

	}
}
