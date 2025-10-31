package builder

import (
	"errors"
	"strconv"

	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/models"
)

var (
	ErrOverrideNotFound = errors.New("override not found in the sandbox")
)

type SandboxBuilder struct {
	internal              models.Sandbox
	err                   error
	lastAddedOverrideName *string
}

type Option func(SandboxBuilder) SandboxBuilder

func NewSandboxBuilder() *SandboxBuilder {
	return &SandboxBuilder{
		internal: models.Sandbox{},
	}
}

func BuildSandbox(name string, opts ...Option) *SandboxBuilder {
	sb := SandboxBuilder{
		internal: models.Sandbox{Name: name},
	}

	for _, opt := range opts {
		sb = opt(sb)
	}

	return &sb
}

func WithData(data models.Sandbox) Option {
	return func(sb SandboxBuilder) SandboxBuilder {
		sb.internal = deepCopy(data)
		// remove deprecated
		sb.internal.Spec.Endpoints = nil
		for _, f := range sb.internal.Spec.Forks {
			f.Endpoints = nil
		}
		return sb
	}
}

// setError sets the error if it hasn't been set already (first error wins)
func (sb *SandboxBuilder) setError(err error) *SandboxBuilder {
	if sb.err == nil && err != nil {
		sb.err = err
	}
	return sb
}

// checkError returns true if there's already an error, false otherwise
func (sb *SandboxBuilder) checkError() bool {
	return sb.err != nil
}

// withError executes a function that can return an error, and sets the error if it fails
func (sb *SandboxBuilder) withError(fn func() error) *SandboxBuilder {
	if sb.checkError() {
		return sb
	}
	return sb.setError(fn())
}

func (sb *SandboxBuilder) Build() (models.Sandbox, error) {
	return sb.internal, sb.err
}

func (sb *SandboxBuilder) GetLastAddedOverrideName() *string {
	return sb.lastAddedOverrideName
}

func (sb *SandboxBuilder) SetMachineID() *SandboxBuilder {
	return sb.withError(func() error {
		machineID, err := system.GetMachineID()
		if err != nil {
			return err
		}
		sb.internal.Spec.LocalMachineID = machineID
		return nil
	})
}

func (sb *SandboxBuilder) AddOverrideMiddleware(worklaodPort int64, toLocal string,
	workloadNames []string, args ...*MiddlewareOverrideArg) *SandboxBuilder {
	if sb.checkError() {
		return sb
	}

	spec := sb.internal.Spec
	if spec.Routing == nil {
		spec.Routing = &models.SandboxesRouting{
			Forwards: make([]*models.SandboxesForward, 0),
		}
	}

	nextForwardIndex := getNextForwardIndex(spec.Routing.Forwards)
	forwardName := "override-" + strconv.Itoa(nextForwardIndex)

	hostArg := &models.SandboxesArgument{
		Name: "overrideHost",
		ValueFrom: &models.SandboxesArgValueFrom{
			Forward: forwardName,
		},
	}

	mw := &models.SandboxesMiddleware{
		Name:  string(OverrideMiddleware),
		Match: make([]*models.SandboxesMiddlewareMatch, 0, len(workloadNames)),
		Args:  []*models.SandboxesArgument{hostArg},
	}

	for _, arg := range args {
		if arg != nil && arg.isSet && arg.internal != nil {
			mw.Args = append(mw.Args, arg.internal(sb, forwardName))
		}
	}

	for _, workloadName := range workloadNames {
		mw.Match = append(mw.Match, &models.SandboxesMiddlewareMatch{
			Workload: workloadName,
		})
	}

	if spec.Middleware == nil {
		spec.Middleware = make([]*models.SandboxesMiddleware, 0)
	}
	spec.Middleware = append(spec.Middleware, mw)

	forwardRouting := &models.SandboxesForward{
		Name:    forwardName,
		Port:    worklaodPort,
		ToLocal: toLocal,
	}

	spec.Routing.Forwards = append(spec.Routing.Forwards, forwardRouting)
	sb.lastAddedOverrideName = &forwardName
	return sb
}

// DeleteOverrideMiddleware deletes the override middleware and the forward by name
// The condition to delete the override is that the overrideName needs to have a valid forward
// and a valid middleware with "overrideHost" with valueFrom.Forward as the arg value
func (sb *SandboxBuilder) DeleteOverrideMiddleware(overrideName string) *SandboxBuilder {
	if sb.checkError() {
		return sb
	}

	// Check if the overrideName is a valid forward name
	if sb.internal.Spec.Routing == nil {
		return sb.setError(ErrOverrideNotFound)
	}
	if !hasOverrideMiddleware(sb.internal.Spec.Middleware, sb.internal.Spec.Routing.Forwards, overrideName) {
		return sb.setError(ErrOverrideNotFound)
	}

	spec := sb.internal.Spec

	// Delete routing.forward by name
	if spec.Routing == nil {
		return sb
	}

	spec.Routing.Forwards = removeForwardByName(spec.Routing.Forwards, overrideName)
	spec.Routing.Forwards = removeForwardByName(spec.Routing.Forwards, getLogForwardName(overrideName))

	// Delete match in the middleware when the args.altHost is the same as the overrideName
	if spec.Middleware == nil {
		return sb
	}

	spec.Middleware = removeMiddleareByValueFrom(spec.Middleware, overrideName)
	return sb
}
