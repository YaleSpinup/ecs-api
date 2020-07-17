package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/YaleSpinup/apierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/gorilla/mux"
)

// TaskShowHandler gets the details for a task in a cluster
func (s *server) TaskShowHandler(w http.ResponseWriter, r *http.Request) {
	w = LogWriter{w}
	vars := mux.Vars(r)
	account := vars["account"]
	cluster := vars["cluster"]
	task := vars["task"]

	ecsService, ok := s.ecsServices[account]
	if !ok {
		msg := fmt.Sprintf("ecs service not found for account: %s", account)
		handleError(w, apierror.New(apierror.ErrNotFound, msg, nil))
		return
	}

	if task == "" {
		handleError(w, apierror.New(apierror.ErrBadRequest, "task cannot be empty", nil))
		return
	}

	output, err := ecsService.GetTasks(r.Context(), &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   aws.StringSlice([]string{task}),
	})
	if err != nil {
		handleError(w, err)
		return
	}

	j, err := json.Marshal(output)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
