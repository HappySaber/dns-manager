package resolvconf

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}
