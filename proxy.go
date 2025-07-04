// Package huefy provides proxy server functionality for other SDKs
// This allows other language SDKs to proxy through the core instead of making direct API calls

package huefy

import (
	"github.com/teracrafts/huefy-sdk/core/kernel"
)

// ProxyRequest re-exports the core ProxyRequest type
type ProxyRequest = core.ProxyRequest

// ProxyConfig re-exports the core ProxyConfig type
type ProxyConfig = core.ProxyConfig

// ProxyResponse re-exports the core ProxyResponse type
type ProxyResponse = core.ProxyResponse

// ProxyError re-exports the core ProxyError type
type ProxyError = core.ProxyError

// ProxyServer re-exports the core ProxyServer type
type ProxyServer = core.ProxyServer

// NewProxyServer re-exports the core constructor
var NewProxyServer = core.NewProxyServer