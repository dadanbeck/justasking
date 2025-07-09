package store

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

// This type of hook separates from the regular PostSave hook since it has side effects
type AfterSaveCommitHook func()

// Hooks for database operations
type Hooks struct {
	PreSave         []func(ctx context.Context, tx *sqlx.Tx, data DTO, isNew bool) error
	PostSave        []func(ctx context.Context, tx *sqlx.Tx, data DTO, model any, isNew bool) error
	PreDelete       []func(ctx context.Context, tx *sqlx.Tx, id int) error
	PostDelete      []func(ctx context.Context, tx *sqlx.Tx, id int) error
	AfterSaveCommit []func(ctx context.Context, data DTO, model any, isNew bool) AfterSaveCommitHook
}

type Datastorer[T any] interface {
	Create(ctx context.Context, data DTO) (any, error)
	Update(ctx context.Context, id int, data DTO) (any, error)
	Delete(ctx context.Context, id int) error
	QueryRow(ctx context.Context, query string, args ...any) (any, error)
	Get(ctx context.Context, query string, args ...any) (*T, error)
	Select(ctx context.Context, query string, args ...any) ([]T, error)

	// WARN: DeleteWhere does not yet support hooks execution.
	DeleteWhere(ctx context.Context, column string, value any) error

	// WARN: BulkUpdate does not run hooks.
	BulkUpdate(ctx context.Context, query string, args ...any) error
	// Set hooks.
	SetHooks(hooks Hooks)

	// useful for complex operations wherein store interface does not supported.
	Base() *sqlx.DB
}

func getStructFieldNamesFromInstance(instance any) []string {
	typ := reflect.TypeOf(instance)
	if typ.Kind() == reflect.Ptr { // Handle pointer types
		typ = typ.Elem()
	}

	var fields []string

	for i := range typ.NumField() {
		field := typ.Field(i)
		dbTag := field.Tag.Get("db")

		if dbTag != "" {
			fields = append(fields, dbTag)
		}
	}

	return fields
}

// getStructFieldsFromDTO extracts field names and placeholders from a DTO struct
func getStructFieldsFromDTO(dto DTO) (columns string, placeholders string) {
	// Get the reflection type of the struct
	t := reflect.TypeOf(dto)
	if t.Kind() == reflect.Ptr {
		t = t.Elem() // Dereference pointer
	}

	var columnNames []string
	var placeholderNames []string

	// Iterate over struct fields
	for i := range t.NumField() {
		field := t.Field(i)

		// Get the `db` tag
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue // Skip fields without a `db` tag or explicitly ignored fields
		}

		columnNames = append(columnNames, dbTag)

		if field.Type.Kind() == reflect.Slice {
			elemType := field.Type.Elem().Kind()
			var pgArrayType string

			switch elemType {
			case reflect.String:
				pgArrayType = "text[]"
			case reflect.Int, reflect.Int32, reflect.Int64:
				pgArrayType = "integer[]"
			case reflect.Float32, reflect.Float64:
				pgArrayType = "float[]"
			case reflect.Bool:
				pgArrayType = "boolean[]"
			default:
				pgArrayType = "text[]"
			}

			placeholderNames = append(placeholderNames, fmt.Sprintf("CAST(:%s AS %s)", dbTag, pgArrayType)) // Named placeholders
		} else {
			placeholderNames = append(placeholderNames, ":"+dbTag) // Named placeholders
		}

	}

	return strings.Join(columnNames, ", "), strings.Join(placeholderNames, ", ")
}

func getNonEmptyFieldsFromDTO(dto DTO, params map[string]any) string {
	v := reflect.ValueOf(dto)
	t := reflect.TypeOf(dto)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	var fields []string

	for i := range v.NumField() {
		field := t.Field(i)
		value := v.Field(i)

		// Check if the field should be skipped entirely
		if field.Tag.Get("db") == "-" {
			continue
		}

		// Convert field names to SQL column names (assumes struct tag `db:"column_name"`)
		columnName := field.Tag.Get("db")
		if columnName == "" {
			columnName = strings.ToLower(field.Name)
		}

		// Skip empty fields
		if value.Kind() == reflect.Ptr && value.IsNil() || value.Kind() == reflect.String && value.String() == "" {
			continue
		}

		if field.Type.Kind() == reflect.Slice {
			elemType := field.Type.Elem().Kind()
			var pgArrayType string

			switch elemType {
			case reflect.String:
				pgArrayType = "text[]"
			case reflect.Int, reflect.Int32, reflect.Int64:
				pgArrayType = "integer[]"
			case reflect.Float32, reflect.Float64:
				pgArrayType = "float[]"
			case reflect.Bool:
				pgArrayType = "boolean[]"
			default:
				pgArrayType = "text[]"
			}

			fields = append(fields, fmt.Sprintf("%s = CAST(:%s AS %s)", columnName, columnName, pgArrayType)) // Named placeholders
		} else {
			fields = append(fields, fmt.Sprintf("%s = :%s", columnName, columnName))
		}
		params[columnName] = value.Interface()
	}

	return strings.Join(fields, ", ")
}
