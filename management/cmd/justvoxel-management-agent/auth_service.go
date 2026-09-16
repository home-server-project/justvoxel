package main

import (
	"errors"
	"fmt"

	"github.com/home-server-project/justvoxel/management/internal/systemauth"
)

type administratorAuthResult struct {
	Mode                   authMode
	PasswordChangeRequired bool
}

type passwordPolicyView struct {
	MinLength int
}

var (
	systemAuthenticate   = systemauth.Authenticate
	systemChangePassword = systemauth.ChangePassword
	systemPasswordPolicy = systemauth.Policy
	systemValidatePass   = systemauth.ValidatePassword
)

func authenticateAdministrator(username, password string) (administratorAuthResult, error) {
	mode, err := currentAuthMode()
	if err != nil {
		return administratorAuthResult{}, err
	}
	if username != systemAdminUsername {
		return administratorAuthResult{}, systemauth.ErrInvalidCredentials
	}

	switch mode {
	case authModeSystem:
		result, err := systemAuthenticate(username, password)
		if err != nil {
			return administratorAuthResult{}, err
		}
		return administratorAuthResult{
			Mode:                   mode,
			PasswordChangeRequired: result.PasswordChangeRequired,
		}, nil
	case authModeSeparate:
		if !verifyLocalAdministrator(username, password) {
			return administratorAuthResult{}, systemauth.ErrInvalidCredentials
		}
		return administratorAuthResult{Mode: mode}, nil
	default:
		return administratorAuthResult{}, fmt.Errorf("unsupported authentication mode %q", mode)
	}
}

func changeAdministratorPassword(mode authMode, currentPassword, newPassword string) error {
	if currentPassword == "" || newPassword == "" {
		return errors.New("current and new passwords are required")
	}

	switch mode {
	case authModeSystem:
		if err := systemValidatePass(systemAdminUsername, currentPassword, newPassword); err != nil {
			return err
		}
		return systemChangePassword(systemAdminUsername, currentPassword, newPassword)
	case authModeSeparate:
		if !verifyLocalAdministrator(systemAdminUsername, currentPassword) {
			return systemauth.ErrInvalidCredentials
		}
		if err := systemValidatePass(systemAdminUsername, currentPassword, newPassword); err != nil {
			return err
		}
		return writeLocalAdministrator(newPassword)
	default:
		return fmt.Errorf("unsupported authentication mode %q", mode)
	}
}

func administratorPasswordPolicy() (passwordPolicyView, error) {
	policy, err := systemPasswordPolicy()
	if err != nil {
		return passwordPolicyView{}, err
	}
	return passwordPolicyView{MinLength: policy.MinLength}, nil
}
