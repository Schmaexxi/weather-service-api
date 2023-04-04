package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/katiamach/weather-service-api/internal/model"

	"github.com/tj/assert"

	mock "github.com/katiamach/weather-service-api/internal/transport/rest/handler/mock"
)

func TestGetWindInfoHandler(t *testing.T) {
	req := &model.WindRequest{
		City: "Stuttgart",
		Year: 2011,
	}

	cases := []struct {
		name           string
		request        *model.WindRequest
		expectedStatus int
		expectedError  error
		isMockCalled   bool
	}{
		{
			name:           "ok",
			request:        req,
			expectedStatus: http.StatusOK,
			isMockCalled:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockWeatherService := mock.NewMockWeatherService(ctrl)
			s := NewWeatherServer(mockWeatherService)

			reqBody, err := json.Marshal(tc.request)
			assert.Nil(t, err)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/wind", bytes.NewReader(reqBody))

			s.GetWindInfoHandler(w, r)

			code := w.Result().StatusCode
			assert.Equal(t, tc.expectedStatus, code)

			var resBody errorResponse
			err = json.NewDecoder(w.Result().Body).Decode(&resBody)
			assert.Nil(t, err)
			defer func() {
				err := w.Result().Body.Close()
				assert.Nil(t, err)
			}()
		})
	}
}
