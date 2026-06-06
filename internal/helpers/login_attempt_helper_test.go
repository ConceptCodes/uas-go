package helpers

import (
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLoginAttemptGetProgressiveDelay(t *testing.T) {
	log := zerolog.New(io.Discard)
	h := NewLoginAttemptHelper(nil, &log)

	tests := []struct {
		name     string
		attempts int64
		expected time.Duration
	}{
		{name: "first attempt", attempts: 0, expected: 0 * time.Second},
		{name: "second attempt", attempts: 1, expected: 2 * time.Second},
		{name: "third attempt", attempts: 2, expected: 5 * time.Second},
		{name: "fourth attempt", attempts: 3, expected: 15 * time.Second},
		{name: "fifth attempt", attempts: 4, expected: 60 * time.Second},
		{name: "sixth attempt max", attempts: 5, expected: 60 * time.Second},
		{name: "tenth attempt max", attempts: 10, expected: 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := h.GetProgressiveDelay(tt.attempts)
			assert.Equal(t, tt.expected, delay)
		})
	}
}
