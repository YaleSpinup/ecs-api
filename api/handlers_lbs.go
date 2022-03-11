package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/ecs-api/elbv2"
	"github.com/YaleSpinup/ecs-api/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Target struct {
	Id     string
	Port   string
	State  string
	Reason string
}

type TargetGroup struct {
	TargetGroupArn  string
	TargetGroupName string
	TargetType      string
	Targets         []Target
}

type ListenerRule struct {
	RuleArn      string
	If           string
	Then         string
	TargetGroups []TargetGroup `json:",omitempty"`
}

type LoadBalancerListener struct {
	ListenerArn  string
	ListenerName string
	SslPolicy    string `json:",omitempty"`
	Rules        []ListenerRule
}

type LoadBalancerSummary struct {
	LoadBalancerArn  string
	LoadBalancerName string
	LoadBalancerType string
	DNSName          string
	Listeners        []LoadBalancerListener
}

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

// LoadBalancerDescribeHandler retuns information about load balancer(s) in a given space
// get ALBs per space (need to be tagged properly)
// get Listeners per ALB (usually 80 and 443)
// get Rules per Listener (get conditions and actions - will contain TG ARN)
// for each TG get the target health and backend servers
// TODO: all this logic should be moved to the orchestrator package
func (s *server) LoadBalancerDescribeHandler(w http.ResponseWriter, r *http.Request) {
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

	log.Infof("getting load balancers in space %s", space)

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

	lbArns, err := rgTaggingAPIService.GetResourcesWithTags(r.Context(), []string{"elasticloadbalancing:loadbalancer"}, tagFilters)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to get resources from the resourcegroupstaggingapi service"))
		return
	}

	out := []LoadBalancerSummary{}
	if len(lbArns) > 0 {
		lbOut, err := elbv2Service.GetLoadBalancers(r.Context(), lbArns)
		if err != nil {
			handleError(w, errors.Wrap(err, "unable to get load balancer details from the elbv2 service"))
			return
		}

		for _, lb := range lbOut {
			lsnrSummary, err := getLBListeners(r.Context(), elbv2Service, aws.StringValue(lb.LoadBalancerArn))
			if err != nil {
				handleError(w, errors.Wrap(err, "unable to get load balancer listeners"))
				return
			}

			out = append(out, LoadBalancerSummary{
				LoadBalancerArn:  aws.StringValue(lb.LoadBalancerArn),
				LoadBalancerName: aws.StringValue(lb.LoadBalancerName),
				LoadBalancerType: aws.StringValue(lb.Type),
				DNSName:          aws.StringValue(lb.DNSName),
				Listeners:        lsnrSummary,
			})
		}
	}

	j, err := json.Marshal(out)
	if err != nil {
		handleError(w, errors.Wrap(err, "unable to marshal response from the elbv2 service"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}

// getLBListeners performs some basic orchestration to get information about load balancer listeners
// such as rules and any associated target groups
func getLBListeners(ctx context.Context, svc elbv2.ELBV2API, arn string) ([]LoadBalancerListener, error) {
	listeners := []LoadBalancerListener{}

	lsnrOut, err := svc.GetListeners(ctx, arn)
	if err != nil {
		return nil, err
	}

	for _, lsnr := range lsnrOut {
		rulesSummary, err := getListenerRules(ctx, svc, aws.StringValue(lsnr.ListenerArn))
		if err != nil {
			return nil, err
		}

		listeners = append(listeners, LoadBalancerListener{
			ListenerArn:  aws.StringValue(lsnr.ListenerArn),
			ListenerName: fmt.Sprintf("%s:%d", aws.StringValue(lsnr.Protocol), aws.Int64Value(lsnr.Port)),
			SslPolicy:    aws.StringValue(lsnr.SslPolicy),
			Rules:        rulesSummary,
		})
	}

	return listeners, nil
}

func getListenerRules(ctx context.Context, svc elbv2.ELBV2API, arn string) ([]ListenerRule, error) {
	rules := []ListenerRule{}
	rulesOut, err := svc.GetRules(ctx, arn)
	if err != nil {
		return nil, err
	}

	for _, r := range rulesOut {
		var ruleIf, ruleThen string
		var tgs []TargetGroup

		// build If description from rule conditions
		if aws.BoolValue(r.IsDefault) {
			ruleIf = "Default"
		} else {
			if len(r.Conditions) > 0 {
				for _, cond := range r.Conditions {
					switch aws.StringValue(cond.Field) {
					case "host-header":
						ruleIf = fmt.Sprintf("%s: %s", aws.StringValue(cond.Field), strings.Join(aws.StringValueSlice(cond.HostHeaderConfig.Values), " or "))
					case "path-pattern":
						ruleIf = fmt.Sprintf("%s: %s", aws.StringValue(cond.Field), strings.Join(aws.StringValueSlice(cond.PathPatternConfig.Values), " or "))
					case "http-header":
						ruleIf = fmt.Sprintf("%s: %s", aws.StringValue(cond.Field), strings.Join(aws.StringValueSlice(cond.HttpHeaderConfig.Values), " or "))
					case "http-request-method":
						ruleIf = fmt.Sprintf("%s: %s", aws.StringValue(cond.Field), strings.Join(aws.StringValueSlice(cond.HttpRequestMethodConfig.Values), " or "))
					case "source-ip":
						ruleIf = fmt.Sprintf("%s: %s", aws.StringValue(cond.Field), strings.Join(aws.StringValueSlice(cond.SourceIpConfig.Values), " or "))
					default:
						ruleIf = aws.StringValue(cond.Field)
					}
				}
			}
		}

		// build Then description from rule actions (and list of TGs when type is forward)
		if len(r.Actions) > 0 {
			for _, act := range r.Actions {
				switch aws.StringValue(act.Type) {
				case "fixed-response":
					ruleThen = fmt.Sprintf("fixed response: status code %s", aws.StringValue(act.FixedResponseConfig.StatusCode))
				case "redirect":
					ruleThen = fmt.Sprintf("redirect to %s://%s:%s%s?%s",
						aws.StringValue(act.RedirectConfig.Protocol),
						aws.StringValue(act.RedirectConfig.Host),
						aws.StringValue(act.RedirectConfig.Port),
						aws.StringValue(act.RedirectConfig.Path),
						aws.StringValue(act.RedirectConfig.Query),
					)
				case "forward":
					var tgNames []string

					for _, tg := range act.ForwardConfig.TargetGroups {
						tgOut, err := svc.GetTargetGroups(ctx, []string{aws.StringValue(tg.TargetGroupArn)})
						if err != nil {
							return nil, err
						}

						if len(tgOut) == 1 {
							var targets []Target

							targetsOut, err := svc.GetTargetHealth(ctx, aws.StringValue(tg.TargetGroupArn))
							if err != nil {
								return nil, err
							}

							for _, t := range targetsOut {
								// a reason and description should be provided if state is not healthy
								reason := aws.StringValue(t.TargetHealth.Reason)
								if t.TargetHealth.Description != nil {
									reason += fmt.Sprintf(" (%s)", aws.StringValue(t.TargetHealth.Description))
								}

								targets = append(targets, Target{
									Id:     aws.StringValue(t.Target.Id),
									Port:   strconv.Itoa(int(aws.Int64Value(t.Target.Port))),
									State:  aws.StringValue(t.TargetHealth.State),
									Reason: reason,
								})
							}

							tgNames = append(tgNames, aws.StringValue(tgOut[0].TargetGroupName))

							tgs = append(tgs, TargetGroup{
								TargetGroupArn:  aws.StringValue(tg.TargetGroupArn),
								TargetGroupName: aws.StringValue(tgOut[0].TargetGroupName),
								TargetType:      aws.StringValue(tgOut[0].TargetType),
								Targets:         targets,
							})
						}
					}

					ruleThen = fmt.Sprintf("forward to %s", strings.Join(tgNames, ", "))
				default:
					ruleThen = aws.StringValue(act.Type)
				}
			}
		}

		rules = append(rules, ListenerRule{
			RuleArn:      aws.StringValue(r.RuleArn),
			If:           ruleIf,
			Then:         ruleThen,
			TargetGroups: tgs,
		})
	}

	return rules, nil
}
