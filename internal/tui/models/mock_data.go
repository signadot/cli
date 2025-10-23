package models

import (
	"fmt"
	"math/rand"
	"time"
)

// MockDataService provides mock HTTP request data for testing
type MockDataService struct {
	requests []HTTPRequest
}

// NewMockDataService creates a new mock data service
func NewMockDataService() *MockDataService {
	service := &MockDataService{
		requests: generateMockRequests(),
	}
	return service
}

// GetRequests returns mock HTTP requests
func (m *MockDataService) GetRequests() []HTTPRequest {
	return m.requests
}

// GetRequestByID returns a specific request by ID
func (m *MockDataService) GetRequestByID(id string) *HTTPRequest {
	for _, req := range m.requests {
		if req.ID == id {
			return &req
		}
	}
	return nil
}

// generateMockRequests creates sample HTTP requests for testing
func generateMockRequests() []HTTPRequest {
	now := time.Now()
	requests := []HTTPRequest{
		{
			ID:                  "req-001",
			MiddlewareRequestID: "181d0181",
			Method:              "GET",
			URL:                 "https://api.example.com/users",
			Path:                "/users",
			RequestURI:          "http://location.hotrod-devmesh.svc:8081/locations",
			RoutingKey:          "zdxbwcdpfz0sl",
			NormHost:            "location.hotrod-devmesh.svc.cluster.local:8081",
			DestWorkload:        "Deployment/hotrod-devmesh/location",
			Proto:               "HTTP/1.1",
			WatchOptions:        "+stream/+stream",
			When:                now.Add(-5 * time.Minute),
			DoneAt:              func() *time.Time { t := now.Add(-5 * time.Minute).Add(150 * time.Millisecond); return &t }(),
			Headers:             map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token123"},
			Body:                "",
			Timestamp:           now.Add(-5 * time.Minute),
			Duration:            150 * time.Millisecond,
			StatusCode:          200,
			ClientIP:            "192.168.1.100",
			UserAgent:           "curl/8.16.0",
			Response: &HTTPResponse{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json", "Content-Length": "1024"},
				Body:       `{"users": [{"id": 1, "name": "John Doe"}]}`,
				Size:       1024,
				Duration:   150 * time.Millisecond,
			},
		},
		{
			ID:                  "req-002",
			MiddlewareRequestID: "50e5d9ce",
			Method:              "GET",
			URL:                 "https://api.example.com/users",
			Path:                "/users",
			RequestURI:          "http://location.hotrod-devmesh.svc:8081/locations",
			RoutingKey:          "zdxbwcdpfz0sl",
			NormHost:            "location.hotrod-devmesh.svc.cluster.local:8081",
			DestWorkload:        "Deployment/hotrod-devmesh/location",
			Proto:               "HTTP/1.1",
			WatchOptions:        "+stream/+stream",
			When:                now.Add(-4 * time.Minute),
			DoneAt:              func() *time.Time { t := now.Add(-4 * time.Minute).Add(300 * time.Millisecond); return &t }(),
			Headers:             map[string]string{"Content-Type": "application/json"},
			Body:                `{"name": "Jane Smith", "email": "jane@example.com"}`,
			Timestamp:           now.Add(-4 * time.Minute),
			Duration:            300 * time.Millisecond,
			StatusCode:          201,
			ClientIP:            "192.168.1.101",
			UserAgent:           "curl/8.16.0",
			Response: &HTTPResponse{
				StatusCode: 201,
				Headers:    map[string]string{"Content-Type": "application/json", "Location": "/users/2"},
				Body:       `{"id": 2, "name": "Jane Smith", "email": "jane@example.com"}`,
				Size:       512,
				Duration:   300 * time.Millisecond,
			},
		},
		{
			ID:                  "req-003",
			MiddlewareRequestID: "de5c2000",
			Method:              "POST",
			URL:                 "https://api.example.com/users/1",
			Path:                "/users/1",
			RequestURI:          "http://location.hotrod-devmesh.svc:8081/locations",
			RoutingKey:          "zdxbwcdpfz0sl",
			NormHost:            "location.hotrod-devmesh.svc.cluster.local:8081",
			DestWorkload:        "Deployment/hotrod-devmesh/location",
			Proto:               "HTTP/1.1",
			WatchOptions:        "+stream/+stream",
			When:                now.Add(-3 * time.Minute),
			DoneAt:              func() *time.Time { t := now.Add(-3 * time.Minute).Add(200 * time.Millisecond); return &t }(),
			Headers:             map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token123"},
			Body:                `{"name": "John Updated", "email": "john.updated@example.com"}`,
			Timestamp:           now.Add(-3 * time.Minute),
			Duration:            200 * time.Millisecond,
			StatusCode:          200,
			ClientIP:            "192.168.1.100",
			UserAgent:           "curl/8.16.0",
			Response: &HTTPResponse{
				StatusCode: 200,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"id": 1, "name": "John Updated", "email": "john.updated@example.com"}`,
				Size:       768,
				Duration:   200 * time.Millisecond,
			},
		},
		{
			ID:                  "req-004",
			MiddlewareRequestID: "a1b2c3d4",
			Method:              "DELETE",
			URL:                 "https://api.example.com/users/2",
			Path:                "/users/2",
			RequestURI:          "http://location.hotrod-devmesh.svc:8081/locations",
			RoutingKey:          "zdxbwcdpfz0sl",
			NormHost:            "location.hotrod-devmesh.svc.cluster.local:8081",
			DestWorkload:        "Deployment/hotrod-devmesh/location",
			Proto:               "HTTP/1.1",
			WatchOptions:        "+stream/+stream",
			When:                now.Add(-2 * time.Minute),
			DoneAt:              func() *time.Time { t := now.Add(-2 * time.Minute).Add(100 * time.Millisecond); return &t }(),
			Headers:             map[string]string{"Authorization": "Bearer token123"},
			Body:                "",
			Timestamp:           now.Add(-2 * time.Minute),
			Duration:            100 * time.Millisecond,
			StatusCode:          204,
			ClientIP:            "192.168.1.102",
			UserAgent:           "curl/8.16.0",
			Response: &HTTPResponse{
				StatusCode: 204,
				Headers:    map[string]string{},
				Body:       "",
				Size:       0,
				Duration:   100 * time.Millisecond,
			},
		},
		{
			ID:                  "req-005",
			MiddlewareRequestID: "e5f6g7h8",
			Method:              "GET",
			URL:                 "https://api.example.com/users/999",
			Path:                "/users/999",
			RequestURI:          "http://location.hotrod-devmesh.svc:8081/locations",
			RoutingKey:          "zdxbwcdpfz0sl",
			NormHost:            "location.hotrod-devmesh.svc.cluster.local:8081",
			DestWorkload:        "Deployment/hotrod-devmesh/location",
			Proto:               "HTTP/1.1",
			WatchOptions:        "+stream/+stream",
			When:                now.Add(-1 * time.Minute),
			DoneAt:              func() *time.Time { t := now.Add(-1 * time.Minute).Add(50 * time.Millisecond); return &t }(),
			Headers:             map[string]string{"Authorization": "Bearer token123"},
			Body:                "",
			Timestamp:           now.Add(-1 * time.Minute),
			Duration:            50 * time.Millisecond,
			StatusCode:          404,
			ClientIP:            "192.168.1.103",
			UserAgent:           "curl/8.16.0",
			Response: &HTTPResponse{
				StatusCode: 404,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"error": "User not found"}`,
				Size:       256,
				Duration:   50 * time.Millisecond,
			},
		},
	}

	for i := 0; i < 10; i++ {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		paths := []string{"/api/users", "/api/orders", "/api/products", "/api/auth/login", "/api/health"}
		statusCodes := []int{200, 201, 400, 401, 403, 404, 500}
		routingKeys := []string{"zdxbwcdpfz0sl", "abc123def456", "xyz789uvw012", "mno345pqr678"}
		requestURIs := []string{
			"http://location.hotrod-devmesh.svc:8081/locations",
			"http://frontend.hotrod-devmesh.svc:8080/api/users",
			"http://backend.hotrod-devmesh.svc:8082/api/orders",
			"http://auth.hotrod-devmesh.svc:8083/login",
		}

		req := HTTPRequest{
			ID:                  fmt.Sprintf("req-%03d", i+6),
			MiddlewareRequestID: generateRandomID(),
			Method:              methods[rand.Intn(len(methods))],
			URL:                 "https://api.example.com" + paths[rand.Intn(len(paths))],
			Path:                paths[rand.Intn(len(paths))],
			RequestURI:          requestURIs[rand.Intn(len(requestURIs))],
			RoutingKey:          routingKeys[rand.Intn(len(routingKeys))],
			NormHost:            "location.hotrod-devmesh.svc.cluster.local:8081",
			DestWorkload:        "Deployment/hotrod-devmesh/location",
			Proto:               "HTTP/1.1",
			WatchOptions:        "+stream/+stream",
			When:                now.Add(-time.Duration(rand.Intn(60)) * time.Minute),
			DoneAt: func() *time.Time {
				t := now.Add(-time.Duration(rand.Intn(60)) * time.Minute).Add(time.Duration(rand.Intn(1000)) * time.Millisecond)
				return &t
			}(),
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       generateRandomBody(),
			Timestamp:  now.Add(-time.Duration(rand.Intn(60)) * time.Minute),
			Duration:   time.Duration(rand.Intn(1000)) * time.Millisecond,
			StatusCode: statusCodes[rand.Intn(len(statusCodes))],
			ClientIP:   fmt.Sprintf("192.168.1.%d", 100+rand.Intn(50)),
			UserAgent:  "curl/8.16.0",
		}

		if req.StatusCode >= 200 && req.StatusCode < 300 {
			req.Response = &HTTPResponse{
				StatusCode: req.StatusCode,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"success": true}`,
				Size:       int64(rand.Intn(1000)),
				Duration:   req.Duration,
			}
		}

		requests = append(requests, req)
	}

	return requests
}

// generateRandomBody creates a random JSON body for mock requests
func generateRandomBody() string {
	bodies := []string{
		`{"name": "Test User"}`,
		`{"email": "test@example.com", "password": "secret"}`,
		`{"query": "search term"}`,
		`{"id": 123, "status": "active"}`,
		`{"data": {"key": "value"}}`,
		"",
	}
	return bodies[rand.Intn(len(bodies))]
}

// generateRandomID creates a random middleware request ID
func generateRandomID() string {
	chars := "0123456789abcdef"
	id := make([]byte, 8)
	for i := range id {
		id[i] = chars[rand.Intn(len(chars))]
	}
	return string(id)
}
