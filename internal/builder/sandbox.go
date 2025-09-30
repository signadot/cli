package builder

import (
	"strconv"
	"strings"

	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/models"
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

func (sb *SandboxBuilder) Build() (models.Sandbox, error) {
	return sb.internal, sb.err
}

// Error returns the current error, if any
func (sb *SandboxBuilder) Error() error {
	return sb.err
}

// HasError returns true if there's an error, false otherwise
func (sb *SandboxBuilder) HasError() bool {
	return sb.checkError()
}

type MiddlewareName string

const (
	AltflowMiddleware MiddlewareName = "altflow"
)

func (sb *SandboxBuilder) AddOverrideMiddleware(worklaodPort int64, toLocal string, workloadNames ...string) *SandboxBuilder {
	if sb.checkError() {
		return sb
	}

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

	sb.lastAddedOverrideName = &forwardName

	return sb
}

func (sb *SandboxBuilder) DeleteOverrideMiddleware(overrideName string) *SandboxBuilder {
	if sb.checkError() {
		return sb
	}

	spec := sb.internal.Spec

	// Delete routing.forward by name
	if spec.Routing == nil {
		return sb
	}

	spec.Routing.Forwards = removeForwardByName(spec.Routing.Forwards, overrideName)

	// Delete match in the middleware when the args.altHost is the same as the overrideName
	if spec.Middleware == nil {
		return sb
	}

	spec.Middleware = removeMiddleareByValueFrom(spec.Middleware, overrideName)

	return sb
}

func removeForwardByName(forwards []*models.SandboxesForward, name string) []*models.SandboxesForward {
	newForwards := make([]*models.SandboxesForward, 0)
	for _, forward := range forwards {
		if forward.Name != name {
			newForwards = append(newForwards, forward)
		}
	}
	return newForwards
}

func removeMiddleareByValueFrom(middlewares []*models.SandboxesMiddleware, value string) []*models.SandboxesMiddleware {
	newMiddlewares := make([]*models.SandboxesMiddleware, 0)

	getAltHostArg := func(middleware *models.SandboxesMiddleware) *models.SandboxesArgument {
		for _, arg := range middleware.Args {
			if arg.Name == "altHost" {
				return arg
			}
		}
		return nil
	}

	for _, middleware := range middlewares {
		altHostArg := getAltHostArg(middleware)

		// TODO: Replace the value with "valueFrom" argument
		if altHostArg != nil && altHostArg.Value == value {
			continue
		}

		newMiddlewares = append(newMiddlewares, middleware)
	}
	return newMiddlewares
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
