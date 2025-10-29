package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/signadot/cli/internal/utils/system"
	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/override"
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

type MiddlewareOverrideArg struct {
	internal func(sb *SandboxBuilder, overrideName string) *models.SandboxesArgument
	isSet    bool
}

func NewOverrideArgPolicy(excludedStatusCodes []int) (*MiddlewareOverrideArg, error) {
	policy := override.Policy{}
	if len(excludedStatusCodes) > 0 {
		policy.OverrideByDefault = true
		policy.ExcludedStatusCodes = excludedStatusCodes
	}

	policyValue, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	applyInternal := func(sb *SandboxBuilder, overrideName string) *models.SandboxesArgument {
		return &models.SandboxesArgument{
			Name:  "policy",
			Value: string(policyValue),
		}
	}

	return &MiddlewareOverrideArg{
		internal: applyInternal,
		isSet:    true,
	}, nil
}

func getLogForwardName(overrideName string) string {
	return fmt.Sprintf("%s-log", overrideName)
}

func NewOverrideLogArg(logListenerPort int) (*MiddlewareOverrideArg, error) {
	arg := &models.SandboxesArgument{
		Name: "logHost",
		ValueFrom: &models.SandboxesArgValueFrom{
			Forward: "logHost",
		},
	}

	applyInternal := func(sb *SandboxBuilder, overrideName string) *models.SandboxesArgument {
		routing := &models.SandboxesForward{
			Name:    getLogForwardName(overrideName),
			Port:    7777,
			ToLocal: "localhost:" + strconv.FormatInt(int64(logListenerPort), 10),
		}

		sb.internal.Spec.Routing.Forwards = append(sb.internal.Spec.Routing.Forwards, routing)
		arg.ValueFrom.Forward = routing.Name

		return arg
	}

	return &MiddlewareOverrideArg{
		internal: applyInternal,
		isSet:    true,
	}, nil
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

type DetailedOverrideMiddleware struct {
	Forward    *models.SandboxesForward
	LogForward *models.SandboxesForward
}

// GetAvailableOverrideMiddlewares returns all available override forwards from a sandbox
func GetAvailableOverrideMiddlewares(sandbox models.Sandbox) []*DetailedOverrideMiddleware {
	var overrides []*DetailedOverrideMiddleware

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

		// Find the forward referenced by this middleware
		for _, arg := range middleware.Args {
			if arg.Name == "overrideHost" && arg.ValueFrom != nil && arg.ValueFrom.Forward != "" {
				forwardName := arg.ValueFrom.Forward
				forward, exists := forwardMap[forwardName]
				if !exists {
					continue
				}

				logForwardName := getLogForwardName(forwardName)
				logForward := forwardMap[logForwardName]

				overrides = append(overrides, &DetailedOverrideMiddleware{
					Forward:    forward,
					LogForward: logForward,
				})
			}
		}
	}

	return overrides
}

// HasOverrideMiddleware checks if a specific override middleware exists by name
func HasOverrideMiddleware(sandbox models.Sandbox, overrideName string) bool {
	overrides := GetAvailableOverrideMiddlewares(sandbox)
	for _, override := range overrides {
		if override.Forward.Name == overrideName {
			return true
		}
	}
	return false
}

// hasOverrideMiddleware is a legacy function that maintains backward compatibility
func hasOverrideMiddleware(middlewares []*models.SandboxesMiddleware, forwards []*models.SandboxesForward, overrideName string) bool {
	// Create a temporary sandbox structure for the new function
	sandbox := models.Sandbox{
		Spec: &models.SandboxSpec{
			Middleware: middlewares,
			Routing: &models.SandboxesRouting{
				Forwards: forwards,
			},
		},
	}
	return HasOverrideMiddleware(sandbox, overrideName)
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
		// remove deprecated
		sb.internal.Spec.Endpoints = nil
		for _, f := range sb.internal.Spec.Forks {
			f.Endpoints = nil
		}
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
