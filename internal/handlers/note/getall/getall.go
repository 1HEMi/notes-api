package getall

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

type AllNoteGetter interface {
	GetAllNotes(userID, limit, offset int, sort string) ([]models.Note, error)
}

func GetUserID(r *http.Request) (int, bool) {
	uid := JWTMiddleware.GetUserID(r.Context())
	if uid == 0 {
		return 0, false
	}
	return uid, true
}

func New(log *slog.Logger, allNoteGetter AllNoteGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.note.getall.New"

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
			render.JSON(w, r, response.Error("forbidden access"))
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

		notes, err := allNoteGetter.GetAllNotes(userIDFromToken, limit, offset, sort)
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
		log.Info("notes was delivered successfully")
		render.JSON(w, r, notes)

	}
}
