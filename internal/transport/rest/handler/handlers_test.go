package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/katiamach/weather-service-api/internal/model"

	"github.com/tj/assert"

	mock "github.com/katiamach/weather-service-api/internal/transport/rest/handler/mock"
)

var errTest = errors.New("test error")

func TestGetWindInfoHandler(t *testing.T) {
	ctx := context.Background()

	req := &model.WindRequest{}

	cases := []struct {
		name           string
		request        *model.WindRequest
		expectedStatus int
		expectedError  error
		isMockCalled   bool
	}{
		{
			name:           "service error",
			request:        req,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  errTest,
			isMockCalled:   true,
		},
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

			if tc.isMockCalled {
				mockWeatherService.EXPECT().
					GetWindInfo(ctx, tc.request).
					Return(tc.expectedError)
			}

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
