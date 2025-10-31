package builder

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/signadot/go-sdk/models"
	"github.com/signadot/libconnect/common/override"
)

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

type DetailedOverrideMiddleware struct {
	Forward    *models.SandboxesForward
	LogForward *models.SandboxesForward
}

// GetAvailableOverrideMiddlewares returns all available override forwards from a sandbox
func GetAvailableOverrideMiddlewares(sandbox *models.Sandbox) []*DetailedOverrideMiddleware {
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
func HasOverrideMiddleware(sb *models.Sandbox, overrideName string) bool {
	overrides := GetAvailableOverrideMiddlewares(sb)
	for _, override := range overrides {
		if override.Forward.Name == overrideName {
			return true
		}
	}
	return false
}

// hasOverrideMiddleware is a legacy function that maintains backward compatibility
func hasOverrideMiddleware(middlewares []*models.SandboxesMiddleware,
	forwards []*models.SandboxesForward, overrideName string) bool {
	// Create a temporary sandbox structure for the new function
	sandbox := &models.Sandbox{
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
