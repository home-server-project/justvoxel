package main

import "errors"

type switchMatrixFixtureStage string

const (
	matrixFixtureConfiguration switchMatrixFixtureStage = "runtime_configuration"
	matrixFixtureCredential    switchMatrixFixtureStage = "credential_validation"
	matrixFixtureStorage       switchMatrixFixtureStage = "storage_placement"
	matrixFixtureSoftware      switchMatrixFixtureStage = "software_version"
	matrixFixturePorts         switchMatrixFixtureStage = "port_reservation"
	matrixFixtureOwnership     switchMatrixFixtureStage = "fixture_ownership"
	matrixFixturePublication   switchMatrixFixtureStage = "runtime_publication"
	matrixFixtureBinding       switchMatrixFixtureStage = "fixture_binding"
)

// Keep the original cause for internal recovery decisions. Only the fixed
// stage and reason selected below belong in public diagnostics.
type switchMatrixFixtureError struct {
	stage switchMatrixFixtureStage
	cause error
}

func (e *switchMatrixFixtureError) Error() string { return string(e.stage) }
func (e *switchMatrixFixtureError) Unwrap() error { return e.cause }

func wrapSwitchMatrixFixtureError(stage switchMatrixFixtureStage, err error) error {
	if err == nil {
		return nil
	}
	var failure *switchMatrixFixtureError
	if errors.As(err, &failure) {
		return err
	}
	if errors.Is(err, errInstanceRuntimeCredential) {
		stage = matrixFixtureCredential
	}
	return &switchMatrixFixtureError{stage: stage, cause: err}
}
