package systemauth

/*
#cgo LDFLAGS: -lpwquality
#include <stdlib.h>
#include <pwquality.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	pam "github.com/msteinert/pam/v2"
)

const pamService = "justvoxel"

var (
	ErrInvalidCredentials = errors.New("invalid system credentials")
	ErrAccountUnavailable = errors.New("system account unavailable")
)

type AuthResult struct {
	PasswordChangeRequired bool
}

type PasswordPolicy struct {
	MinLength int
}

// Authenticate verifies a local system account through the JustVoxel PAM
// service. The caller decides which system usernames are allowed to administer
// the appliance; this package deliberately does not broaden that policy.
func Authenticate(username, password string) (AuthResult, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tx, err := pam.StartFunc(pamService, username, func(style pam.Style, _ string) (string, error) {
		switch style {
		case pam.PromptEchoOff:
			return password, nil
		case pam.PromptEchoOn:
			return username, nil
		case pam.ErrorMsg, pam.TextInfo:
			return "", nil
		default:
			return "", pam.ErrConv
		}
	})
	if err != nil {
		return AuthResult{}, fmt.Errorf("start PAM transaction: %w", err)
	}
	defer func() { _ = tx.End() }()

	if err := tx.Authenticate(pam.DisallowNullAuthtok); err != nil {
		return AuthResult{}, fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
	}
	if err := tx.AcctMgmt(pam.DisallowNullAuthtok); err != nil {
		if errors.Is(err, pam.ErrNewAuthtokReqd) {
			return AuthResult{PasswordChangeRequired: true}, nil
		}
		return AuthResult{}, fmt.Errorf("%w: %v", ErrAccountUnavailable, err)
	}
	return AuthResult{}, nil
}

// Policy returns the effective libpwquality settings from the host. This is
// intentionally read from AlmaLinux/RHEL configuration rather than duplicated
// as a JustVoxel-specific password policy.
func Policy() (PasswordPolicy, error) {
	settings, err := loadPWQualitySettings()
	if err != nil {
		return PasswordPolicy{}, err
	}
	defer C.pwquality_free_settings(settings)

	var minLength C.int
	if rc := C.pwquality_get_int_value(settings, C.PWQ_SETTING_MIN_LENGTH, &minLength); rc < 0 {
		return PasswordPolicy{}, pwqualityError(rc, nil)
	}
	return PasswordPolicy{MinLength: int(minLength)}, nil
}

// ValidatePassword checks a proposed password against the host's current
// libpwquality configuration. PAM remains authoritative for actually changing
// the system password; this check lets the WebUI explain the same host policy
// without inventing its own composition rules.
func ValidatePassword(username, oldPassword, newPassword string) error {
	settings, err := loadPWQualitySettings()
	if err != nil {
		return err
	}
	defer C.pwquality_free_settings(settings)

	newC := C.CString(newPassword)
	defer C.free(unsafe.Pointer(newC))
	oldC := C.CString(oldPassword)
	defer C.free(unsafe.Pointer(oldC))
	userC := C.CString(username)
	defer C.free(unsafe.Pointer(userC))

	var aux unsafe.Pointer
	if rc := C.pwquality_check(settings, newC, oldC, userC, &aux); rc < 0 {
		return pwqualityError(rc, aux)
	}
	return nil
}

func loadPWQualitySettings() (*C.pwquality_settings_t, error) {
	settings := C.pwquality_default_settings()
	if settings == nil {
		return nil, errors.New("libpwquality could not allocate settings")
	}
	var aux unsafe.Pointer
	if rc := C.pwquality_read_config(settings, nil, &aux); rc != 0 {
		err := pwqualityError(rc, aux)
		C.pwquality_free_settings(settings)
		return nil, err
	}
	return settings, nil
}

func pwqualityError(code C.int, aux unsafe.Pointer) error {
	message := C.pwquality_strerror(nil, 0, code, aux)
	if message == nil {
		return fmt.Errorf("libpwquality rejected password (code %d)", int(code))
	}
	return errors.New(C.GoString(message))
}
