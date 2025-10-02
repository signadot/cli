package builder

import (
	"errors"
	"strconv"
	"strings"

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
	OverrideMiddleware MiddlewareName = "override"
)

func (sb *SandboxBuilder) AddOverrideMiddleware(worklaodPort int64, toLocal string, workloadNames ...string) *SandboxBuilder {
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
	if !hasOverrideMiddleware(sb.internal.Spec.Middleware, sb.internal.Spec.Routing.Forwards, overrideName) {
		return sb.setError(ErrOverrideNotFound)
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

func hasOverrideMiddleware(middlewares []*models.SandboxesMiddleware, forwards []*models.SandboxesForward, overrideName string) bool {
	filteredMiddlewares := make([]*models.SandboxesMiddleware, 0)
	for _, middleware := range middlewares {
		if middleware.Name == string(OverrideMiddleware) {
			filteredMiddlewares = append(filteredMiddlewares, middleware)
		}
	}

	if len(filteredMiddlewares) == 0 {
		return false
	}

	middlewareMetForward := false
	// Check if any middleware contains the overrideName as the valueFrom.Forward
	for _, middleware := range filteredMiddlewares {
		for _, arg := range middleware.Args {
			if arg.ValueFrom != nil && arg.ValueFrom.Forward == overrideName {
				middlewareMetForward = true
				break
			}
		}
	}

	if !middlewareMetForward {
		return false
	}

	// Check if any routing.forward has the overrideName as the name
	for _, forward := range forwards {
		if forward.Name == overrideName {
			return true
		}
	}

	return false
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
	var filteredMiddlewares []*models.SandboxesMiddleware

	for _, middleware := range middlewares {
		if !hasOverrideHostArgWithValue(middleware, value) {
			filteredMiddlewares = append(filteredMiddlewares, middleware)
		}
	}

	return filteredMiddlewares
}

// hasOverrideHostArgWithValue checks if a middleware has an overrideHost argument
// that references the specified forward value
func hasOverrideHostArgWithValue(middleware *models.SandboxesMiddleware, forwardValue string) bool {
	for _, arg := range middleware.Args {

		if arg.Name != "overrideHost" {
			continue
		}

		if arg.ValueFrom == nil {
			continue
		}

		if arg.ValueFrom.Forward == forwardValue {
			return true
		}
	}
	return false
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

// GetAvailableOverrideMiddlewares returns all available override middlewares from a sandbox
func GetAvailableForwardForOverrideMiddlewares(sandbox models.Sandbox) []*models.SandboxesForward {
	var overrides []*models.SandboxesForward

	// Check if sandbox has middleware and routing
	if sandbox.Spec.Middleware == nil || sandbox.Spec.Routing == nil || sandbox.Spec.Routing.Forwards == nil {
		return overrides
	}

	// Create a map of forwards for quick lookup
	forwardMap := make(map[string]*models.SandboxesForward)
	for _, forward := range sandbox.Spec.Routing.Forwards {
		forwardMap[forward.Name] = forward
	}

	// Find all override middlewares
	for _, middleware := range sandbox.Spec.Middleware {
		if middleware.Name != string(OverrideMiddleware) {
			continue
		}

		// Extract workload names from middleware matches
		var workloads []string
		for _, match := range middleware.Match {
			if match.Workload != "" {
				workloads = append(workloads, match.Workload)
			}
		}

		// Find the forward referenced by this middleware
		for _, arg := range middleware.Args {
			if arg.Name == "overrideHost" && arg.ValueFrom != nil && arg.ValueFrom.Forward != "" {
				forwardName := arg.ValueFrom.Forward
				if forward, exists := forwardMap[forwardName]; exists {
					overrides = append(overrides, &models.SandboxesForward{
						Name:        forwardName,
						Port:        forward.Port,
						ToLocal:     forward.ToLocal,
						AppProtocol: forward.AppProtocol,
					})
				}
			}
		}
	}

	return overrides
}
