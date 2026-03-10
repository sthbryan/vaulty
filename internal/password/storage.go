package password

type Storage interface {
	Get() (string, error)
	Set(password string) error
	Delete() error
	Type() string
}
