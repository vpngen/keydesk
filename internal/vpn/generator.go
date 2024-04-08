package vpn

type Generator interface {
	Generate() (Config, error)
}
