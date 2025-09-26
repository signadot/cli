package builder

import (
	"strconv"
	"strings"

	"github.com/signadot/go-sdk/models"
)

type SandboxBuilder struct {
	internal models.Sandbox
}

type Option func(SandboxBuilder) SandboxBuilder

func BuildSandbox(name string, opts ...Option) *SandboxBuilder {
	sb := SandboxBuilder{
		internal: models.Sandbox{Name: name},
	}

	for _, opt := range opts {
		sb = opt(sb)
	}

	sb.internal.Name = "overridden"

	return &sb
}

func (sb *SandboxBuilder) Build() models.Sandbox {
	return sb.internal
}

type MiddlewareName string

const (
	AltflowMiddleware MiddlewareName = "altflow"
)

func (sb *SandboxBuilder) AddOverrideMiddleware(worklaodPort int64, toLocal string, workloadNames ...string) string {
	hostArg := &models.SandboxesArgument{
		Name:  "altHost",
		Value: sb.internal.Name,
	}

	mw := &models.SandboxesMiddleware{
		Name:  string(AltflowMiddleware),
		Match: make([]*models.SandboxesMiddlewareMatch, 0, len(workloadNames)),
		Args:  []*models.SandboxesArgument{hostArg},
	}

	for _, workloadName := range workloadNames {
		mw.Match = append(mw.Match, &models.SandboxesMiddlewareMatch{
			Workload: workloadName,
		})
	}

	spec := sb.internal.Spec
	if spec.Middleware == nil {
		spec.Middleware = make([]*models.SandboxesMiddleware, 0)
	}

	spec.Middleware = append(spec.Middleware, mw)

	if spec.Routing == nil {
		spec.Routing = &models.SandboxesRouting{
			Forwards: make([]*models.SandboxesForward, 0),
		}
	}

	nextForwardIndex := getNextForwardIndex(spec.Routing.Forwards)
	forwardName := "override-" + strconv.Itoa(nextForwardIndex)
	forwardRouting := &models.SandboxesForward{
		Name:    forwardName,
		Port:    worklaodPort,
		ToLocal: toLocal,
	}

	spec.Routing.Forwards = append(spec.Routing.Forwards, forwardRouting)

	return forwardName
}

func WithData(data models.Sandbox) Option {
	return func(sb SandboxBuilder) SandboxBuilder {
		sb.internal = deepCopy(data)
		return sb
	}
}

func getNextForwardIndex(forwards []*models.SandboxesForward) int {
	maxIndex := getMaxForwardIndex(forwards)
	return maxIndex + 1
}

func getMaxForwardIndex(forwards []*models.SandboxesForward) int {
	// This is assuming that the forward name is in the format "override-<index>"
	maxIndex := -1
	for _, forward := range forwards {
		if strings.HasPrefix(forward.Name, "override-") {
			index, err := strconv.Atoi(strings.TrimPrefix(forward.Name, "override-"))
			if err != nil {
				continue
			}

			if index > maxIndex {
				maxIndex = index
			}
		}
	}
	return maxIndex
}
