package grafana

import (
	// Standard library.
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"text/template"

	// Internal packages.
	"go.deuill.org/webhook-gateway/pkg/gateway"
)

func TestNew(t *testing.T) {
	var testCases = []struct {
		descr   string
		options []Option
		err     error
	}{
		{
			descr: "new instance with no options",
		},
		{
			descr: "new instance with malformed template",
			options: []Option{
				WithTemplate(`Hello {{name}}!`),
			},
			err: errors.New(`failed parsing message template: template: message:1: function "name" not defined`),
		},
		{
			descr: "new instance with correct template",
			options: []Option{
				WithTemplate(`Hello {{.Name}}!`),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.descr, func(t *testing.T) {
			_, err := New(tt.options...)
			if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) {
				t.Fatalf("New(): want error '%v', have '%v'", tt.err, err)
			} else if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Fatalf("New(): want error '%s', have '%s'", tt.err.Error(), err.Error())
			}
		})
	}
}

func TestGrafanaParseTemplate(t *testing.T) {
	var testCases = []struct {
		descr   string
		source  *Grafana
		request *http.Request

		expect []*gateway.Message
		err    error
	}{
		{
			descr:  "authentication failure for missing secret value",
			source: &Grafana{},
			request: httptest.NewRequestWithContext(
				gateway.SetSecret(context.Background(), "1234"),
				"POST", "/test", nil,
			),
			err: errors.New("Authorization header not found"),
		},
		{
			descr:  "authentication failure for malformed Bearer token",
			source: &Grafana{},
			request: func() *http.Request {
				req := httptest.NewRequestWithContext(
					gateway.SetSecret(context.Background(), "1234"),
					"POST", "/test", nil,
				)
				req.Header.Set("Authorization", "1234")
				return req
			}(),
			err: errors.New("invalid Bearer token"),
		},
		{
			descr:  "authentication failure for incorrect Bearer token",
			source: &Grafana{},
			request: func() *http.Request {
				req := httptest.NewRequestWithContext(
					gateway.SetSecret(context.Background(), "1234"),
					"POST", "/test", nil,
				)
				req.Header.Set("Authorization", "Bearer 123")
				return req
			}(),
			err: errors.New("invalid Bearer token"),
		},
		{
			descr:  "authorization success",
			source: &Grafana{},
			request: func() *http.Request {
				req := httptest.NewRequestWithContext(
					gateway.SetSecret(context.Background(), "1234"),
					"POST", "/test", nil,
				)
				req.Header.Set("Authorization", "Bearer 1234")
				return req
			}(),
			err: errors.New("failed parsing request: unexpected end of JSON input"),
		},
		{
			descr:  "authorization passthrough without secret",
			source: &Grafana{},
			request: func() *http.Request {
				req := httptest.NewRequest("POST", "/test", nil)
				req.Header.Set("Authorization", "Bearer foobar")
				return req
			}(),
			err: errors.New("failed parsing request: unexpected end of JSON input"),
		},
		{
			descr:   "invalid JSON body",
			source:  &Grafana{},
			request: httptest.NewRequest("POST", "/test", strings.NewReader("{what?}")),
			err:     errors.New("failed parsing request: invalid character 'w' looking for beginning of object key string"),
		},
		{
			descr:   "no payload content",
			source:  &Grafana{},
			request: httptest.NewRequest("POST", "/test", strings.NewReader(`{"status": "firing"}`)),
			err:     errors.New("no message content found"),
		},
		{
			descr: "template execution failure",
			source: &Grafana{template: func() *template.Template {
				tpl, _ := template.New("message").Parse("Alert! Alert! {{.Foo}}")
				return tpl
			}()},
			request: httptest.NewRequest("POST", "/test", strings.NewReader(`{"status": "firing"}`)),
			err:     errors.New(`template: message:1:16: executing "message" at <.Foo>: can't evaluate field Foo in type grafana.Payload`),
		},
		{
			descr: "message from template",
			source: &Grafana{template: func() *template.Template {
				tpl, _ := template.New("message").Parse("Alert! Alert! {{.Status}}")
				return tpl
			}()},
			request: httptest.NewRequest("POST", "/test", strings.NewReader(`{"status": "firing"}`)),
			expect:  []*gateway.Message{{Content: "Alert! Alert! firing"}},
		},
		{
			descr:   "message from content",
			source:  &Grafana{},
			request: httptest.NewRequest("POST", "/test", strings.NewReader(`{"title": "Hello", "message": "World"}`)),
			expect:  []*gateway.Message{{Content: "Hello\nWorld"}},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.descr, func(t *testing.T) {
			msg, err := tt.source.ParseHTTP(tt.request)
			if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) {
				t.Fatalf("Grafana.ParseMessage(): want error '%v', have '%v'", tt.err, err)
			} else if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Fatalf("Grafana.ParseMessage(): want error '%s', have '%s'", tt.err.Error(), err.Error())
			} else if !reflect.DeepEqual(msg, tt.expect) {
				t.Fatalf("Grafana.ParseMessage(): want message '%#v', have '%#v'", tt.expect, msg)
			}
		})
	}
}

func TestGrafanaUnmarshalTOML(t *testing.T) {
	var testCases = []struct {
		descr string
		data  any

		expect *Grafana
		err    error
	}{
		{
			descr:  "no data",
			expect: &Grafana{},
		},
		{
			descr:  "data with invalid type",
			data:   42,
			expect: &Grafana{},
		},
		{
			descr: "data with unknown fields",
			data: map[string]any{
				"foo": "bar",
			},
			expect: &Grafana{},
		},
		{
			descr: "data with invalid template field",
			data: map[string]any{
				"template": "{{here}}",
			},
			err:    errors.New(`failed parsing message template: template: message:1: function "here" not defined`),
			expect: &Grafana{},
		},
		{
			descr: "data with valid template field",
			data: map[string]any{
				"template": "{{.Foo}}",
			},
			expect: &Grafana{
				template: func() *template.Template {
					tpl, _ := template.New("message").Parse("{{.Foo}}")
					return tpl
				}(),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.descr, func(t *testing.T) {
			g := &Grafana{}
			err := g.UnmarshalTOML(tt.data)
			if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) {
				t.Fatalf("Grafana.UnmarshalTOML(): want error '%v', have '%v'", tt.err, err)
			} else if err != nil && tt.err != nil && err.Error() != tt.err.Error() {
				t.Fatalf("Grafana.UnmarshalTOML(): want error '%s', have '%s'", tt.err.Error(), err.Error())
			} else if !reflect.DeepEqual(g, tt.expect) {
				t.Fatalf("Grafana.ParseMessage(): want gateway '%#v', have '%#v'", tt.expect, g)
			}
		})
	}
}
