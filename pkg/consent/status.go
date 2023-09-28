package consent

type Status int

const (
	NotAsked = iota
	Accepted
	Declined
	Expired
)

func (s Status) String() string {
	return [...]string{"not-asked", "accepted", "declined", "declined"}[s]
}
