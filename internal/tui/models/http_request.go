package models

import (
	"time"
)

// HTTPRequest represents a captured HTTP request
type HTTPRequest struct {
	ID                  string            `json:"id"`
	MiddlewareRequestID string            `json:"middlewareRequestID"`
	Method              string            `json:"method"`
	URL                 string            `json:"url"`
	Path                string            `json:"path"`
	RequestURI          string            `json:"requestURI"`
	RoutingKey          string            `json:"routingKey"`
	NormHost            string            `json:"normHost"`
	DestWorkload        string            `json:"destWorkload"`
	Proto               string            `json:"proto"`
	WatchOptions        string            `json:"watchOptions"`
	When                time.Time         `json:"when"`
	DoneAt              *time.Time        `json:"doneAt,omitempty"`
	Headers             map[string]string `json:"headers"`
	Body                string            `json:"body"`
	Timestamp           time.Time         `json:"timestamp"`
	Duration            time.Duration     `json:"duration"`
	StatusCode          int               `json:"status_code"`
	Response            *HTTPResponse     `json:"response,omitempty"`
	ClientIP            string            `json:"client_ip"`
	UserAgent           string            `json:"user_agent"`
}

// HTTPResponse represents the response to an HTTP request
type HTTPResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Size       int64             `json:"size"`
	Duration   time.Duration     `json:"duration"`
}

// HTTPRequestList represents a list of HTTP requests
type HTTPRequestList struct {
	Requests []HTTPRequest `json:"requests"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PerPage  int           `json:"per_page"`
}

// GetStatusColor returns the color for the status code
func (r *HTTPRequest) GetStatusColor() string {
	if r.StatusCode >= 200 && r.StatusCode < 300 {
		return "green"
	} else if r.StatusCode >= 300 && r.StatusCode < 400 {
		return "yellow"
	} else if r.StatusCode >= 400 && r.StatusCode < 500 {
		return "orange"
	} else if r.StatusCode >= 500 {
		return "red"
	}
	return "gray"
}

// GetMethodColor returns the color for the HTTP method
func (r *HTTPRequest) GetMethodColor() string {
	switch r.Method {
	case "GET":
		return "blue"
	case "POST":
		return "green"
	case "PUT":
		return "yellow"
	case "DELETE":
		return "red"
	case "PATCH":
		return "purple"
	default:
		return "gray"
	}
}

// FormatDuration returns a formatted duration string
func (r *HTTPRequest) FormatDuration() string {
	if r.Duration < time.Millisecond {
		return "< 1ms"
	} else if r.Duration < time.Second {
		return r.Duration.Round(time.Millisecond).String()
	} else {
		return r.Duration.Round(time.Millisecond).String()
	}
}

// GetShortURL returns a shortened version of the URL for display
func (r *HTTPRequest) GetShortURL() string {
	url := r.URL
	if r.RequestURI != "" {
		url = r.RequestURI
	}

	return url
}

// GetShortRequestURI returns a shortened version of the request URI for display
func (r *HTTPRequest) GetShortRequestURI() string {
	return r.RequestURI
}

// GetShortRoutingKey returns a shortened version of the routing key for display
func (r *HTTPRequest) GetShortRoutingKey() string {
	return r.RoutingKey
}

// GetShortPath returns a shortened version of the path for display
func (r *HTTPRequest) GetShortPath() string {
	return r.Path
}
