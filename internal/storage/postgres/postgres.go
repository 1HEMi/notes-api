package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"notes/internal/models"
	"notes/internal/storage"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Storage struct {
	db *sql.DB
}

func New(StoragePath string) (*Storage, error) {
	const op = "storage.postgres.New"
	db, err := sql.Open("postgres", StoragePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: ping: %w", op, err)
	}

	return &Storage{
		db: db,
	}, nil
}

func (s *Storage) SaveUser(username, password string) (int, error) {
	const op = "storage.postgres.SaveUser"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("%s: hash password: %w", op, err)
	}
	var userID int
	err = s.db.QueryRow(
		"INSERT INTO users(username, password) VALUES($1, $2) RETURNING id",
		username, hashedPassword,
	).Scan(&userID)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return 0, storage.ErrUserExists
		}
		return 0, fmt.Errorf("%s: insert user: %w", op, err)
	}

	return userID, nil
}

func (s *Storage) GetUserByUsername(username string) (*models.User, error) {
	const op = "storage.postgres.GetUserByUsername"

	stmt, err := s.db.Prepare("SELECT id, username, password, created_at FROM users WHERE username=$1")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()
	var u models.User
	err = stmt.QueryRow(username).Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}
	return &u, nil
}

func (s *Storage) SaveNote(userID int, title, content string) error {
	const op = "storage.postgres.SaveNote"
	stmt, err := s.db.Prepare("INSERT INTO notes(user_id, title, content) VALUES($1, $2, $3)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	res, err := stmt.Exec(userID, title, content)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	_ = res
	return nil
}

func (s *Storage) GetNote(userID, noteID int) (*models.Note, error) {
	const op = "storage.postgres.GetNote"
	stmt, err := s.db.Prepare("SELECT id, user_id, title, content, created_at, updated_at FROM notes WHERE id=$1 AND user_id=$2")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()
	var resNote models.Note
	err = stmt.QueryRow(noteID, userID).Scan(
		&resNote.ID,
		&resNote.UserID,
		&resNote.Title,
		&resNote.Content,
		&resNote.CreatedAt,
		&resNote.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%s: query row: %w", op, err)
	}
	return &resNote, nil
}

func (s *Storage) GetAllNotes(userID, limit, offset int, sort string) ([]models.Note, error) {
	const op = "storage.postgres.GetAllNotes"
	if sort != "asc" && sort != "desc" {
		sort = "desc"
	}
	query := `
		SELECT id, user_id, title, content, created_at, updated_at
		FROM notes
		WHERE user_id = $1
		ORDER BY created_at ` + sort + `
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()
	var notes []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%s: scan: %w", op, err)
		}
		notes = append(notes, n)
	}
	if len(notes) == 0 {
		return nil, storage.ErrNoteNotFound
	}
	return notes, nil
}

func (s *Storage) UpdateNote(noteID int, userID int, title, content string) error {
	const op = "storage.postgres.UpdateNote"
	var ownerID int
	err := s.db.QueryRow("SELECT user_id FROM notes WHERE id=$1", noteID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.ErrNoteNotFound
		}
		return fmt.Errorf("%s: query row: %w", op, err)
	}
	if ownerID != userID {
		return storage.ErrForbidden
	}
	stmt, err := s.db.Prepare("UPDATE notes SET title=$1, content=$2, updated_at=NOW() WHERE id=$3")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	res, err := stmt.Exec(title, content, noteID)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}
	_ = res
	return nil
}

func (s *Storage) DeleteNote(noteID, userID int) error {
	const op = "storage.postgres.DeleteNote"
	var ownerID int
	err := s.db.QueryRow("SELECT user_id FROM notes WHERE id=$1", noteID).Scan(&ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return storage.ErrNoteNotFound
		}
		return fmt.Errorf("%s: query row: %w", op, err)
	}
	if ownerID != userID {
		return storage.ErrForbidden
	}
	_, err = s.db.Exec("DELETE FROM notes WHERE id=$1", noteID)
	if err != nil {
		return fmt.Errorf("%s: delete exec: %w", op, err)
	}
	return nil
}
