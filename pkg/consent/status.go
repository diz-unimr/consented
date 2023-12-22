package consent

type Status int

const (
	NotAsked = iota
	Accepted
	Declined
	Expired
	Withdrawn
)

func (s Status) String() string {
	return [...]string{"not-asked", "accepted", "declined", "expired", "withdrawn"}[s]
}
