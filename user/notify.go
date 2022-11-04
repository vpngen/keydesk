package user

import "net/netip"

const (
	NotifyNewBrigade = "new-brigade"
	NotifyDelBrigade = "del-brigade"
	NotifyNewUser    = "new-user"
	NotifyDelUser    = "del-user"
)

type SrvBrigade struct {
	ID          string `json:"id"`
	IdentityKey []byte `json:"key"`
}

type SrvUser struct {
	ID          string `json:"id,omitempty"`
	WgPublicKey []byte `json:"wg_pub,omitempty"`
	IsBrigadier bool   `json:"boss,omitempty"`
}

type SrvNotify struct {
	T        string     `json:"type"`
	Endpoint string     `json:"endpoint"`
	Brigade  SrvBrigade `json:"brigade"`
	User     SrvUser    `json:"user,omitempty"`
}

func NewEndpoint(addr netip.Addr) string {
	buf := [16]byte{0xfd, 0xcc, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x03}
	copy(buf[2:6], addr.AsSlice())

	return netip.AddrFrom16(buf).String()
}
