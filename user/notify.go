package user

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
	T       string     `json:"type"`
	Brigade SrvBrigade `json:"brigade"`
	User    SrvUser    `json:"user,omitempty"`
}
