package store

type DTO interface {
	ToModel(id int) any
}
