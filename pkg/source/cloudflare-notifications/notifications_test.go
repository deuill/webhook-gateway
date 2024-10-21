package cloudflare_notifications

import (
	// Standard library.
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	// Internal packages.
	"go.deuill.org/webhook-gateway/pkg/gateway"
)

func TestNotificationsParseTemplate(t *testing.T) {
	var testCases = []struct {
		descr   string
		source  *Notifications
		request *http.Request

		expect []*gateway.Message
		err    error
	}{
		{
			descr:  "authentication failure for missing secret value",
			source: &Notifications{},
			request: httptest.NewRequestWithContext(
				gateway.SetSecret(context.Background(), "1234"),
				"POST", "/test", nil,
			),
			err: errors.New("cf-webhook-auth header not found"),
		},
		{
			descr:  "authentication failure for incorrect authentication token",
			source: &Notifications{},
			request: func() *http.Request {
				req := httptest.NewRequestWithContext(
					gateway.SetSecret(context.Background(), "1234"),
					"POST", "/test", nil,
				)
				req.Header.Set("cf-webhook-auth", "123")
				return req
			}(),
			err: errors.New("invalid authentication token"),
		},
		{
			descr:  "authorization success",
			source: &Notifications{},
			request: func() *http.Request {
				req := httptest.NewRequestWithContext(
					gateway.SetSecret(context.Background(), "1234"),
					"POST", "/test", nil,
				)
				req.Header.Set("cf-webhook-auth", "1234")
				return req
			}(),
			err: errors.New("failed parsing request: unexpected end of JSON input"),
		},
		{
			descr:  "authorization passthrough without secret",
			source: &Notifications{},
			request: func() *http.Request {
				req := httptest.NewRequest("POST", "/test", nil)
				req.Header.Set("cf-webhook-auth", "foobar")
				return req
			}(),
			err: errors.New("failed parsing request: unexpected end of JSON input"),
		},
		{
			descr:   "invalid JSON body",
			source:  &Notifications{},
			request: httptest.NewRequest("POST", "/test", strings.NewReader("{what?}")),
			err:     errors.New("failed parsing request: invalid character 'w' looking for beginning of object key string"),
		},
		{
			descr:   "no payload content",
			source:  &Notifications{},
			request: httptest.NewRequest("POST", "/test", strings.NewReader(`{"foo": "bar"}`)),
			err:     errors.New("no message content found"),
		},
		{
			descr:   "message from content",
			source:  &Notifications{},
			request: httptest.NewRequest("POST", "/test", strings.NewReader(`{"text": "Hello World"}`)),
			expect:  []*gateway.Message{{Content: "Hello World"}},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.descr, func(t *testing.T) {
			msg, err := tt.source.ParseHTTP(tt.request)
			if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) {
				t.Fatalf("Notifications.ParseMessage(): want error '%v', have '%v'", tt.err, err)
			} else if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Fatalf("Notifications.ParseMessage(): want error '%s', have '%s'", tt.err.Error(), err.Error())
			} else if !reflect.DeepEqual(msg, tt.expect) {
				t.Fatalf("Notifications.ParseMessage(): want message '%#v', have '%#v'", tt.expect, msg)
			}
		})
	}
}
