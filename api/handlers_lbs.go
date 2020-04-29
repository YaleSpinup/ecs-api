package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/YaleSpinup/ecs-api/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// LoadBalancerListHandler lists the target groups with the appropriate org and spaceid
func (s *server) LoadBalancerListHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	space := vars["space"]

	rgTaggingAPIService, ok := s.rgTaggingAPIServices[account]
	if !ok {
		msg := fmt.Sprintf("resourcegroupstaggingapi service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	elbv2Service, ok := s.elbv2Services[account]
	if !ok {
		msg := fmt.Sprintf("elbv2 service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	log.Infof("getting resources in space %s", space)

	tagFilters := []*resourcegroupstaggingapi.TagFilter{
		{
			Key:   "spinup:org",
			Value: []string{s.org},
		},
		{
			Key:   "spinup:spaceid",
			Value: []string{space},
		},
	}

	arns, err := rgTaggingAPIService.GetResourcesWithTags(r.Context(), []string{"elasticloadbalancing:targetgroup"}, tagFilters)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to get resources from the resourcegroupstaggingapi service"))
		return
	}

	out := make(map[string]string, len(arns))
	if len(arns) > 0 {
		tgOut, err := elbv2Service.GetTargetGroups(r.Context(), arns)
		if err != nil {
			handleError(w, errors.Wrap(err, "unable to get target group details from the elbv2 service"))
			return
		}

		for _, tg := range tgOut {
			out[aws.StringValue(tg.TargetGroupName)] = aws.StringValue(tg.TargetGroupArn)
		}
	}

	j, err := json.Marshal(out)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the ssm service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
