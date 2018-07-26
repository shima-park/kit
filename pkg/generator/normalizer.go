package generator

type Normalizer interface {
	Normalize(name string) string
}

var NoopNormalizer Normalizer = noopNormalizer{}

type noopNormalizer struct {
}

func (n noopNormalizer) Normalize(name string) string {
	return name
}
