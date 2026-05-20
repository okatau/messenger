package domain

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
)

type UserStatus struct {
	Status   string
	Metadata map[string]any
}
