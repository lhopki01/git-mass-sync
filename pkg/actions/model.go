package actions

type Severity int

const (
	Info Severity = iota
	Warning
	Error
)

type Repo struct {
	Name     string `json:"name"`
	SSHURL   string `json:"ssh_url"`
	Message  string
	Severity Severity
	Archived bool `json:"archived"`
}

type Repos []*Repo

func (s Severity) String() string {
	return [...]string{"Info", "Warning", "Error"}[s]
}
