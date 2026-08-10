package delete_test

import (
	"encoding/json"
	"errors"
	"http-service/internal/http-server/handlers/url/delete"
	"http-service/internal/http-server/handlers/url/delete/mocks"
	"http-service/internal/lib/api/response"
	"http-service/internal/lib/logger/handlers/slogdiscard"
	"http-service/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		respError string
		mockError error
	}{
		{
			name:  "Success",
			alias: "test_alias",
		},
		{
			name:      "Not found",
			alias:     "not_found",
			respError: "not found",
			mockError: storage.ErrURLNotFound,
		},
		{
			name:      "DeleteURL error",
			alias:     "test_alias",
			respError: "failed to delete url",
			mockError: errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			urlDeleterMock := mocks.NewURLDeleter(t)

			if tc.respError == "" || tc.mockError != nil {
				urlDeleterMock.
					On("DeleteURL", tc.alias).
					Return(tc.mockError).
					Once()
			}

			handler := delete.New(
				slogdiscard.NewDiscardLogger(),
				urlDeleterMock,
			)

			r := chi.NewRouter()

			r.Delete(
				"/url/{alias}",
				handler,
			)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/url/"+tc.alias,
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

func TestDeleteHandlerEmptyAlias(t *testing.T) {
	urlDeleterMock := mocks.NewURLDeleter(t)

	handler := delete.New(
		slogdiscard.NewDiscardLogger(),
		urlDeleterMock,
	)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/url/",
		nil,
	)

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var resp response.Response

	require.NoError(
		t,
		json.Unmarshal(rr.Body.Bytes(), &resp),
	)

	require.Equal(
		t,
		"alias is required",
		resp.Error,
	)

	require.Equal(
		t,
		http.StatusBadRequest,
		rr.Code,
	)
}
