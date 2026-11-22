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
	"strings"
	"unsafe"

	pam "github.com/msteinert/pam/v2"
)

const pamService = "justvoxel"

var (
	ErrInvalidCredentials = errors.New("invalid system credentials")
	ErrAccountUnavailable = errors.New("system account unavailable")
	ErrPasswordChange     = errors.New("system password change failed")
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

// ChangePassword changes the real local system password through the JustVoxel
// PAM service. Authentication and password-policy enforcement remain owned by
// the AlmaLinux/RHEL PAM stack; plaintext passwords are provided only through
// PAM's in-process conversation callback and are never placed in command-line
// arguments, environment variables, files, or logs.
func ChangePassword(username, currentPassword, newPassword string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	changing := false
	changePrompt := 0
	tx, err := pam.StartFunc(pamService, username, func(style pam.Style, message string) (string, error) {
		switch style {
		case pam.PromptEchoOn:
			return username, nil
		case pam.PromptEchoOff:
			if !changing {
				return currentPassword, nil
			}
			response := passwordChangeResponse(message, changePrompt, currentPassword, newPassword)
			changePrompt++
			return response, nil
		case pam.ErrorMsg, pam.TextInfo:
			return "", nil
		default:
			return "", pam.ErrConv
		}
	})
	if err != nil {
		return fmt.Errorf("%w: start PAM transaction: %v", ErrPasswordChange, err)
	}
	defer func() { _ = tx.End() }()

	if err := tx.Authenticate(pam.DisallowNullAuthtok); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
	}
	if err := tx.AcctMgmt(pam.DisallowNullAuthtok); err != nil && !errors.Is(err, pam.ErrNewAuthtokReqd) {
		return fmt.Errorf("%w: %v", ErrAccountUnavailable, err)
	}

	changing = true
	if err := tx.ChangeAuthTok(0); err != nil {
		return fmt.Errorf("%w: %v", ErrPasswordChange, err)
	}
	return nil
}

// passwordChangeResponse maps standard Linux-PAM password prompts without
// making the WebUI responsible for PAM conversation details. Explicit old-token
// prompts receive the current password; explicit and unknown change prompts
// receive the proposed new password because authentication already succeeded.
func passwordChangeResponse(message string, promptIndex int, currentPassword, newPassword string) string {
	prompt := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(prompt, "current"), strings.Contains(prompt, "old"):
		return currentPassword
	case strings.Contains(prompt, "new"), strings.Contains(prompt, "retype"), strings.Contains(prompt, "again"):
		return newPassword
	default:
		// During pam_chauthtok the caller has already authenticated the user.
		// Some PAM modules therefore omit the old-password prompt and issue a
		// generic or empty first prompt for the new token. Treat unknown change
		// prompts as new-password prompts instead of replaying the old token.
		_ = promptIndex
		return newPassword
	}
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
