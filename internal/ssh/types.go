package ssh

type Host struct {
	Name         string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	ProxyJump    string
}
