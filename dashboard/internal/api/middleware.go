package api

import (
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

// endpoints_registry returns the endpoint template registry for the API.
func endpoints_registry() []endpoints.EndpointTemplate {
	return endpoints.Registry
}
