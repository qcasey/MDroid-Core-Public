package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qcasey/MDroid-Core/format/response"
)

func TestSlackAlert(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/alert", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleSlackAlert)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expectedResponse := response.JSONResponse{Output: "Slack URL not set in config.", OK: false}
	var resp response.JSONResponse
	json.NewDecoder(rr.Body).Decode(&resp)

	if expectedResponse.OK != resp.OK {
		t.Errorf("handler returned unexpected OK: got %v want %v",
			resp.OK, expectedResponse.OK)
	}
}
