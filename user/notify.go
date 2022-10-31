package user

const (
	NotifyNewBrigade = "new-brigade"
	NotifyDelBrigade = "del-brigade"
	NotifyNewUser    = "new-user"
	NotifyDelUser    = "del-user"
)

type SrvBrigadier struct {
	ID          string
	WgPublicKey []byte
}

type SrvUser struct {
	ID          string
	WgPublicKey []byte
	IsBrigadier bool
}

type SrvNotify struct {
	T         string
	Brigadier SrvBrigadier
	User      SrvUser
}
