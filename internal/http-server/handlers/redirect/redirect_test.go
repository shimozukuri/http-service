package redirect_test

import (
	"encoding/json"
	"errors"
	"http-service/internal/lib/api/response"
	"http-service/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"http-service/internal/http-server/handlers/redirect"
	"http-service/internal/http-server/handlers/redirect/mocks"
	"http-service/internal/lib/api"
	"http-service/internal/lib/logger/handlers/slogdiscard"
)

func TestRedirectHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		respError string
		mockError error
	}{
		{
			name:  "Success",
			alias: "test_alias",
			url:   "https://www.google.com/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlGetterMock := mocks.NewURLGetter(t)

			urlGetterMock.
				On("GetURL", tc.alias).
				Return(tc.url, tc.mockError).
				Once()

			r := chi.NewRouter()

			r.Get(
				"/{alias}",
				redirect.New(
					slogdiscard.NewDiscardLogger(),
					urlGetterMock,
				),
			)

			ts := httptest.NewServer(r)
			defer ts.Close()

			redirectedToURL, err := api.GetRedirect(
				ts.URL + "/" + tc.alias,
			)

			require.NoError(t, err)

			require.Equal(
				t,
				tc.url,
				redirectedToURL,
			)
		})
	}
}

func TestRedirectHandlerError(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		respError string
		mockError error
	}{
		{
			name:      "URL not found",
			alias:     "not_found",
			respError: "not found",
			mockError: storage.ErrURLNotFound,
		},
		{
			name:      "GetURL error",
			alias:     "test_alias",
			respError: "internal error",
			mockError: errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlGetterMock := mocks.NewURLGetter(t)

			urlGetterMock.
				On("GetURL", tc.alias).
				Return(tc.url, tc.mockError).
				Once()

			r := chi.NewRouter()

			r.Get(
				"/{alias}",
				redirect.New(
					slogdiscard.NewDiscardLogger(),
					urlGetterMock,
				),
			)

			req := httptest.NewRequest(
				http.MethodGet,
				"/"+tc.alias,
				nil,
			)

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			var resp response.Response

			require.NoError(
				t,
				json.Unmarshal(rr.Body.Bytes(), &resp),
			)

			require.Equal(
				t,
				tc.respError,
				resp.Error,
			)
		})
	}
}
