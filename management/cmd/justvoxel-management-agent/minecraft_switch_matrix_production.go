//go:build !justvoxel_dev

package main

import "net/http"

const switchMatrixDevelopment = false

func registerSwitchMatrixExecutionRoutes(*http.ServeMux, *server) {}
