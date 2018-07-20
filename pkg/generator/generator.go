package generator

type Generator interface {
	Generate() error
}

type noopGenerator struct {
}

func (n noopGenerator) Generate() error {
	return nil
}
