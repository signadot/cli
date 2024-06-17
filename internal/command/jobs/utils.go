package jobs

import (
	"fmt"
	"time"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/artifacts"
	"github.com/signadot/go-sdk/client/jobs"
	"github.com/signadot/go-sdk/models"
	"github.com/xeonx/timeago"
)

func getJob(cfg *config.Job, jobName string) (*models.Job, error) {
	params := jobs.NewGetJobParams().WithOrgName(cfg.Org).WithJobName(jobName)
	resp, err := cfg.Client.Jobs.GetJob(params, nil)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}

func getArtifacts(cfg *config.JobGet, job *models.Job) ([]*models.JobArtifact, error) {
	params := artifacts.NewListJobAttemptArtifactsParams().
		WithOrgName(cfg.Org).
		WithJobAttempt(job.Status.Attempts[0].ID).
		WithJobName(job.Name)

	resp, err := cfg.Client.Artifacts.ListJobAttemptArtifacts(params, nil)
	if err != nil {
		return []*models.JobArtifact{}, nil
	}

	return resp.Payload, nil
}

func isJobPhaseToPrintDefault(ph string) bool {
	if ph == "failed" {
		return false
	}
	if ph == "succeeded" {
		return false
	}
	if ph == "canceled" {
		return false
	}
	return true
}

func getAttemptCreatedAtAndDuration(job *models.Job) (createdAtStr string, durationStr string) {
	var createdAt *time.Time

	if len(job.Status.Attempts) == 0 {
		return "", ""
	}

	attempt := job.Status.Attempts[0]

	if attempt.Phase == "queued" {
		return "", ""
	}

	createdAtRaw := attempt.CreatedAt
	if len(createdAtRaw) != 0 {
		t, err := time.Parse(time.RFC3339, createdAtRaw)
		if err != nil {
			return "", ""
		}

		createdAt = &t
		createdAtStr = timeago.NoMax(timeago.English).Format(t)
	}

	finishedAtRaw := attempt.FinishedAt
	if createdAt != nil && len(finishedAtRaw) != 0 {
		finishedAt, err := time.Parse(time.RFC3339, finishedAtRaw)
		if err != nil {
			return "", ""
		}

		durationTime := finishedAt.Sub(*createdAt)
		durationStr = durationTime.String()
	}

	return createdAtStr, durationStr
}

func getJobEnvironment(job *models.Job) string {
	routingContext := job.Spec.RoutingContext

	switch {
	case routingContext == nil:
	case len(routingContext.Sandbox) > 0:
		return fmt.Sprintf("sandbox=%s", routingContext.Sandbox)
	case len(routingContext.Routegroup) > 0:
		return fmt.Sprintf("routegroup=%s", routingContext.Routegroup)
	}

	return "baseline"
}

func getJobStatus(job *models.Job) string {
	switch job.Status.Attempts[0].Phase {
	case "queued":
		return "Queued"
	case "running":
		return "Running"
	case "failed":
		return "Failed"
	case "succeeded":
		return "Succeeded"
	case "canceled":
		return "Canceled"
	}
	return "Unknown"
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
