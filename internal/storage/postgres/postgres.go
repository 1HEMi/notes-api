package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"notes/internal/models"
	"notes/internal/storage"

	"github.com/lib/pq"
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

func (s *Storage) SaveUser(username string) error {
	const op = "storage.postgres.SaveUser"
	stmt, err := s.db.Prepare("INSERT INTO users(username) VALUES($1)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	res, err := stmt.Exec(username)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrUserExists
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	_ = res
	return nil
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

func (s *Storage) GetAllNotes(userID int) ([]models.Note, error) {
	const op = "storage.postgres.GetAllNotes"
	rows, err := s.db.Query("SELECT id, user_id, title, content, created_at, updated_at FROM notes WHERE user_id=$1 ORDER BY created_at DESC", userID)
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
	return notes, nil
}

func (s *Storage) UpdateNote(noteID int, userID int, title, content string) error {
	const op = "storage.postgres.UpdateNote"
	stmt, err := s.db.Prepare("UPDATE notes SET title=$1, content=$2, updated_at=NOW() WHERE id=$3 AND user_id=$4")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	res, err := stmt.Exec(title, content, noteID, userID)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNoteNotFound
	}
	return nil
}

func (s *Storage) DeleteNote(noteID, userID int) error {
	const op = "storage.postgres.DeleteNote"
	stmt, err := s.db.Prepare("DELETE FROM notes WHERE id=$1 AND user_id=$2")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	res, err := stmt.Exec(noteID, userID)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrNoteNotFound
	}
	return nil
}
