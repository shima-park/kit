package generator

type Generator interface {
	Generate() error
}

var NoopGenerator Generator = noopGenerator{}

type noopGenerator struct {
}

func (n noopGenerator) Generate() error {
	return nil
}
