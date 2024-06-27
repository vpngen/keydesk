package ipsec

const (
	UsernameLen = 16
	PasswordLen = 32
)

// Config implements vpn.Config
type Config struct {
	username, password, host, psk                      string
	routerUser, routerPass, shufflerUser, shufflerPass []byte
}
