// Package aago is the root of the AaGo application framework.
//
// The framework itself lives in the aa and pkg subpackages:
//
//	aa             application bootstrap (App)
//	aa/httpsvr     net/http router, request parsing and DTO responses
//	aa/aconfig     yaml configuration, environments and Consul discovery
//	aa/acontext    request-scoped std context, trace and health helpers
//	aa/ae          business error codes
//	aa/alog        logging interface and default implementation
//	aa/atype       business types and enums
//	pkg            dependency-free utilities
//
// Architecture and usage documentation is in the doc directory.
package aago
