package ecs

// Attachment defines things attached to tasks and services
type Attachment struct {
	Status  string
	Type    string
	Details map[string]string
}

// Container is a container in a task and service
type Container struct {
	ID                string
	Health            string
	Name              string
	Status            string
	TaskID            string
	NetworkInterfaces []*NetworkInterface
}

// ContainerOverride is an override for containers in a task or service
type ContainerOverride struct {
	Command     []string
	Environment map[string]string
}

// NetworkInterface is a network interface attached to a container via a task or service
type NetworkInterface struct {
	AttachmentID string
	ID           string
	MacAddress   string
	PrivateIP    string
	Subnet       string
}
