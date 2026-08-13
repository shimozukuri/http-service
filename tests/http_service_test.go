//go:build functional

package tests

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"

	"http-service/internal/http-server/handlers/url/save"
	"http-service/internal/lib/api"
	"http-service/internal/lib/random"
)

const (
	host   = "localhost:8082"
	scheme = "http"
)

func TestHTTPService_Save(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		alias     string
		status    int
		wantError bool
	}{
		{
			name:      "Valid URL",
			url:       gofakeit.URL(),
			alias:     random.NewRandomString(10),
			status:    http.StatusCreated,
			wantError: false,
		},
		{
			name:      "Generated Alias",
			url:       gofakeit.URL(),
			alias:     "",
			status:    http.StatusCreated,
			wantError: false,
		},
		{
			name:      "Invalid URL",
			url:       "some invalid url",
			alias:     random.NewRandomString(10),
			status:    http.StatusBadRequest,
			wantError: true,
		},
		{
			name:      "Empty URL",
			url:       "",
			alias:     random.NewRandomString(10),
			status:    http.StatusBadRequest,
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := newHttpExpect(t)

			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth("myuser", "mypass").
				Expect().
				Status(tc.status).
				JSON().Object()

			if tc.wantError {
				resp.Value("status").String().IsEqual("ERROR")
				resp.Value("error").String().NotEmpty()
				resp.NotContainsKey("alias")

				return
			}

			alias := tc.alias

			if alias == "" {
				resp.Value("alias").String().NotEmpty()
				alias = resp.Value("alias").String().Raw()
			}

			t.Cleanup(func() {
				e.DELETE("/url/"+alias).
					WithBasicAuth("myuser", "mypass").
					Expect()
			})

			resp.Value("status").String().IsEqual("OK")
			resp.Value("alias").String().IsEqual(alias)
			resp.NotContainsKey("error")
		})
	}
}

func TestHTTPService_DuplicateAlias(t *testing.T) {
	e := newHttpExpect(t)

	u, alias := createURL(t, e)

	resp := e.POST("/url").
		WithJSON(save.Request{
			URL:   u,
			Alias: alias,
		}).
		WithBasicAuth("myuser", "mypass").
		Expect().
		Status(http.StatusConflict).
		JSON().Object()

	resp.Value("status").String().IsEqual("ERROR")
	resp.Value("error").String().NotEmpty()
	resp.NotContainsKey("alias")
}

func TestHTTPService_Delete(t *testing.T) {
	e := newHttpExpect(t)

	_, alias := createURL(t, e)

	resp := e.DELETE("/url/"+alias).
		WithBasicAuth("myuser", "mypass").
		Expect().
		Status(http.StatusOK).
		JSON().Object()

	resp.Value("status").String().IsEqual("OK")
}

func TestHTTPService_DeleteNotFound(t *testing.T) {
	e := newHttpExpect(t)

	resp := e.DELETE("/url/"+random.NewRandomString(10)).
		WithBasicAuth("myuser", "mypass").
		Expect().
		Status(http.StatusNotFound).
		JSON().Object()

	resp.Value("status").String().IsEqual("ERROR")
	resp.Value("error").String().NotEmpty()
}

func TestHTTPService_Redirect(t *testing.T) {
	e := newHttpExpect(t)

	u, alias := createURL(t, e)

	testRedirect(t, alias, u)
}

func TestHTTPService_RedirectNotFound(t *testing.T) {
	alias := random.NewRandomString(10)

	testRedirectNotFound(t, alias)
}

func TestHTTPService_SaveRedirectDelete(t *testing.T) {
	e := newHttpExpect(t)

	u := gofakeit.URL()
	alias := gofakeit.Word() + gofakeit.Word()

	// Save

	resp := e.POST("/url").
		WithJSON(save.Request{
			URL:   u,
			Alias: alias,
		}).
		WithBasicAuth("myuser", "mypass").
		Expect().Status(http.StatusCreated).
		JSON().Object()

	resp.Value("alias").String().IsEqual(alias)

	// Redirect

	testRedirect(t, alias, u)

	// Delete

	respDel := e.DELETE("/url/"+alias).
		WithBasicAuth("myuser", "mypass").
		Expect().Status(http.StatusOK).
		JSON().Object()

	respDel.Value("status").String().IsEqual("OK")

	// Redirect again

	testRedirectNotFound(t, alias)
}

func newHttpExpect(t *testing.T) *httpexpect.Expect {
	t.Helper()

	u := url.URL{
		Scheme: scheme,
		Host:   host,
	}

	return httpexpect.Default(t, u.String())
}

func createURL(t *testing.T, e *httpexpect.Expect) (string, string) {
	t.Helper()

	u := gofakeit.URL()
	alias := random.NewRandomString(10)

	e.POST("/url").
		WithJSON(save.Request{
			URL:   u,
			Alias: alias,
		}).
		WithBasicAuth("myuser", "mypass").
		Expect().
		Status(http.StatusCreated)

	t.Cleanup(func() {
		e.DELETE("/url/"+alias).
			WithBasicAuth("myuser", "mypass").
			Expect()
	})

	return u, alias
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	t.Helper()

	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   alias,
	}

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)

	require.Equal(t, urlToRedirect, redirectedToURL)
}

func testRedirectNotFound(t *testing.T, alias string) {
	t.Helper()

	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   alias,
	}

	_, err := api.GetRedirect(u.String())
	require.ErrorIs(t, err, api.ErrInvalidStatusCode)
}
