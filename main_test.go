package main

import (
	"github.com/kataras/iris/v12/httptest"
	"os"
	"testing"
)

func testRESTService(t *testing.T) {
	app := createApp(os.Stdout)
	e := httptest.New(t, app)

	e.GET("/health/ping").Expect().
		Status(httptest.StatusOK).
		Body().Contains("pong")
	e.GET("/api/greeting/werner").Expect().
		Status(httptest.StatusOK).
		Body().Contains("Hello werner")
}
