package reviewtrigger

type giteeClient interface {
	AddPRLabel(owner, repo string, number int, label string) error
	RemovePRLabel(owner, repo string, number int, label string) error
}
