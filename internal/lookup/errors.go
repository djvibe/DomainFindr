package lookup

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/djvibe/domainfindr/internal/model"
)

func classifyRegistrarError(err error) (transient bool, timedOut bool) {
	if err == nil {
		return false, false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true, true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true, true
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded") {
		return true, true
	}
	if strings.Contains(message, "temporary") || strings.Contains(message, "temporarily unavailable") {
		return true, false
	}

	return false, false
}

func isTransientRegistrarHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func markRegistrarFailure(result model.Result, transient bool, timedOut bool) model.Result {
	result.Transient = transient
	result.TimedOut = timedOut
	return result
}
