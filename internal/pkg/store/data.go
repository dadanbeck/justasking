package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/paulexconde/justasking/internal/pkg/fault"
)

type dataStore[T any] struct {
	db         *sqlx.DB
	tablename  string
	hooks      Hooks
	mu         sync.RWMutex
	dtoFactory func() any
}

func NewDataStore[T any](db *sqlx.DB, tablename string, dtoFactory ...func() any) *dataStore[T] {
	var factory func() any

	if len(dtoFactory) > 0 {
		factory = dtoFactory[0]
	}

	return &dataStore[T]{
		db:         db,
		tablename:  tablename,
		mu:         sync.RWMutex{},
		dtoFactory: factory,
	}
}

func (s *dataStore[T]) Base() *sqlx.DB {
	return s.db
}

func (s *dataStore[T]) SetHooks(hooks Hooks) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hooks.PreSave = append(s.hooks.PreSave, hooks.PreSave...)
	s.hooks.PostSave = append(s.hooks.PostSave, hooks.PostSave...)
	s.hooks.PreDelete = append(s.hooks.PreDelete, hooks.PreDelete...)
	s.hooks.PostDelete = append(s.hooks.PostDelete, hooks.PostDelete...)
}

func (s *dataStore[T]) QueryRow(ctx context.Context, query string, args ...any) (any, error) {
	row := s.db.QueryRowContext(ctx, query, args...)

	var result any

	err := row.Scan(&result)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fault.ErrNotFound
		}
		return nil, err
	}

	return result, nil
}

func (s *dataStore[T]) Get(ctx context.Context, query string, args ...any) (*T, error) {
	var result T

	if err := s.db.GetContext(ctx, &result, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fault.ErrNotFound
		}
		return nil, err
	}

	return &result, nil
}

func (s *dataStore[T]) Select(ctx context.Context, query string, args ...any) ([]T, error) {
	var results []T

	if err := s.db.SelectContext(ctx, &results, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []T{}, nil
		}
		return nil, err
	}

	return results, nil
}

func (s *dataStore[T]) Create(ctx context.Context, data DTO) (any, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, hook := range s.hooks.PreSave {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err := hook(ctx, tx, data, true); err != nil {
			return nil, err
		}
	}

	columns, placeholders := getStructFieldsFromDTO(data)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", s.tablename, columns, placeholders)

	stmt, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var id int
	err = stmt.QueryRowContext(ctx, data).Scan(&id)
	if err != nil {
		// Check for unique constraint violation
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" { // PostgreSQL unique constraint violation code
			return nil, fault.ErrUniqueViolation
		}
		return nil, err
	}

	model := data.ToModel(id)

	for _, hook := range s.hooks.PostSave {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if err := hook(ctx, tx, data, model, true); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return model, nil
}

func (s *dataStore[T]) Update(ctx context.Context, id int, data DTO) (any, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, hook := range s.hooks.PreSave {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err := hook(ctx, tx, data, false); err != nil {
			return nil, err
		}
	}

	params := map[string]any{"id": id}
	setClause := getNonEmptyFieldsFromDTO(data, params)

	if setClause == "" {
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = :id", s.tablename, setClause)

	stmt, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	_, err = stmt.ExecContext(ctx, params)
	if err != nil {
		switch e := err.(type) {
		case *pq.Error:
			switch e.Code {
			case "23505": // Check for unique constraint violation
				return nil, fault.ErrUniqueViolation
			}
		case error:
			if e == sql.ErrNoRows {
				return nil, fault.ErrNotFound
			}
		}
		return nil, err
	}

	updatedModel, err := s.getByIDBase(ctx, id) // Ensure you have this method
	if err != nil {
		return nil, err
	}

	for _, hook := range s.hooks.PostSave {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		if err := hook(ctx, tx, data, updatedModel, false); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return updatedModel, nil
}

func (s *dataStore[T]) DeleteWhere(ctx context.Context, column string, value any) error {
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s=$1", s.tablename, column)

	_, err = s.db.ExecContext(ctx, query, value)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" {
				return fault.ErrForeignKeyViolation
			}
		}
	}

	return tx.Commit()
}

func (s *dataStore[T]) Delete(ctx context.Context, id int) error {
	tx, err := s.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, hook := range s.hooks.PreDelete {
		if err := hook(ctx, tx, id); err != nil {
			return err
		}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id=$1", s.tablename)

	_, err = s.db.ExecContext(ctx, query, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" {
				return fault.ErrForeignKeyViolation
			}
		}
	}

	for _, hook := range s.hooks.PostDelete {
		if err := hook(ctx, tx, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *dataStore[T]) BulkUpdate(ctx context.Context, query string, args ...any) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	_, err = tx.ExecContext(ctx, query, args...)

	return err
}

func (s *dataStore[T]) getByIDBase(ctx context.Context, id int) (any, error) {
	var instance any
	if s.dtoFactory != nil {
		instance = s.dtoFactory() // ✅ Use the DTO if factory exists
	} else {
		instance = new(T) // ✅ Otherwise, use the full model (`T`)
	}

	fields := strings.Join(getStructFieldNamesFromInstance(instance), ", ")
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id=$1", fields, s.tablename)

	if err := s.db.GetContext(ctx, instance, query, id); err != nil {
		return nil, err
	}

	return instance, nil
}
