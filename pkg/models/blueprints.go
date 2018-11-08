package models

// Function Result definition
type FnResult interface {
	GetResult(module string, attr string, index int) ([]string, error)
}