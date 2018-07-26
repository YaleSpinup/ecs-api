package servicediscovery

import (
	"git.yale.edu/spinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"

	log "github.com/sirupsen/logrus"
)

// ServiceDiscovery is the internal service discovery object which holds session
// and configuration information
type ServiceDiscovery struct {
	Service *servicediscovery.ServiceDiscovery
}

// NewSession builds a new aws servicediscovery session
func NewSession(account common.Account) ServiceDiscovery {
	s := ServiceDiscovery{}
	log.Infof("Creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = servicediscovery.New(sess)
	return s
}
