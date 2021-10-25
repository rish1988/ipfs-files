package config

type IpfsOptions struct {
	DBPath             string
	NodeType           NodeType
	BootstrapNodes     []string
	EnableExperimental bool
}

type NodeType string

const (
	Default   NodeType = "default"
	Ephemeral NodeType = "ephemeral"
)
