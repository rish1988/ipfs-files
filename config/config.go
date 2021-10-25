package config

type IpfsOptions struct {
	DBPath             string
	NodeType           NodeType
	BootstrapNodes     []string
	EnableExperimental bool
	*SwarmOptions
}

type SwarmOptions struct {
	Private bool
	Key string
}

type NodeType string

const (
	Default   NodeType = "default"
	Ephemeral NodeType = "ephemeral"
)
