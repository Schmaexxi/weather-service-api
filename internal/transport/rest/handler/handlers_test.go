package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/katiamach/weather-service-api/internal/model"

	"github.com/tj/assert"

	mock "github.com/katiamach/weather-service-api/internal/transport/rest/handler/mock"
)

var errTest = errors.New("test error")

func TestGetWindStatisticsHandler(t *testing.T) {
	ctx := context.Background()

	req := &model.WindRequest{
		City:  "stuttgart",
		Years: 5,
	}

	resp := []*model.WindStatistics{}

	okPath := "/windStats?city=stuttgart&years=5"

	cases := []struct {
		name           string
		path           string
		request        *model.WindRequest
		expectedStatus int
		expectedError  error
		isMockCalled   bool
	}{
		{
			name:           "service error",
			path:           "/windStats",
			request:        req,
			expectedStatus: http.StatusBadRequest,
			expectedError:  errors.New("city parameter not provided in query"),
			isMockCalled:   false,
		},
		{
			name:           "service error",
			path:           okPath,
			request:        req,
			expectedStatus: http.StatusInternalServerError,
			expectedError:  errTest,
			isMockCalled:   true,
		},
		{
			name:           "ok",
			path:           okPath,
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
			r := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewReader(reqBody))

			if tc.isMockCalled {
				mockWeatherService.EXPECT().
					GetWindStatistics(ctx, tc.request).
					Return(resp, tc.expectedError)
			}

			s.GetWindStatisticsHandler(w, r)

			code := w.Result().StatusCode
			assert.Equal(t, tc.expectedStatus, code)

			if tc.name == "ok" {
				var resBody []*model.WindStatistics
				err = json.NewDecoder(w.Result().Body).Decode(&resBody)
				assert.Nil(t, err)
				defer w.Result().Body.Close()

				assert.True(t, reflect.DeepEqual(resp, resBody))
				return
			}

			var resBody errorResponse
			err = json.NewDecoder(w.Result().Body).Decode(&resBody)
			assert.Nil(t, err)
			defer w.Result().Body.Close()

			assert.Equal(t, tc.expectedError.Error(), resBody.Message)
		})
	}
}
