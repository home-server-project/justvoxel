package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"time"
)

type twoFactorRecord struct {
	accountCreated time.Time
	Secret         string   `json:"secret"`
	Source         string   `json:"source"`
	RecoveryHashes []string `json:"recovery_hashes"`
	LastStep       int64    `json:"last_step,omitempty"`
}

const (
	twoFactorSelf          = "self"
	twoFactorAdministrator = "administrator"
)

type twoFactorStatus struct {
	Enabled bool   `json:"two_factor_enabled"`
	Source  string `json:"two_factor_source,omitempty"`
}

// With no separate password, the same local authentication file can carry
// voxel's WebUI-only factor. Password writes preserve this field.
func (s *server) administratorFactorPath() string {
	if s.adminFactorPath != "" {
		return s.adminFactorPath
	}
	return localAuthPath
}

func readAdministratorTwoFactor(path string) (*twoFactorRecord, error) {
	store, err := readLocalAuthStoreAt(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return store.TwoFactor, err
}

func changeAdministratorTwoFactor(path string, previous, next *twoFactorRecord) error {
	dir, err := lockLocalAuthDirectory(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	store, err := readLocalAuthStoreAt(path)
	if errors.Is(err, os.ErrNotExist) {
		store = localAuthStore{Version: localAuthVersion}
	} else if err != nil {
		return err
	}
	if (previous == nil) != (store.TwoFactor == nil) ||
		(previous != nil && previous.Secret != store.TwoFactor.Secret) {
		return errTwoFactorChanged
	}
	store.TwoFactor = next
	return writeLocalAuthStoreAt(path, store)
}

func (s *server) readTwoFactor(userID int64) (*twoFactorRecord, error) {
	if userID == 0 {
		return readAdministratorTwoFactor(s.administratorFactorPath())
	}
	var record twoFactorRecord
	var hashes string
	err := s.store.db.QueryRow(`SELECT secret, source, recovery_hashes, last_step FROM web_user_two_factor WHERE user_id = ?`, userID).
		Scan(&record.Secret, &record.Source, &hashes, &record.LastStep)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(hashes), &record.RecoveryHashes); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *server) changeTwoFactor(userID int64, previous, next *twoFactorRecord) error {
	if userID == 0 {
		return changeAdministratorTwoFactor(s.administratorFactorPath(), previous, next)
	}
	var hashes []byte
	if next != nil {
		var err error
		hashes, err = json.Marshal(next.RecoveryHashes)
		if err != nil {
			return err
		}
	}
	var result sql.Result
	var err error
	switch {
	case next == nil:
		result, err = s.store.db.Exec(`DELETE FROM web_user_two_factor WHERE user_id = ? AND secret = ?`, userID, previous.Secret)
	case previous == nil:
		result, err = s.store.db.Exec(`INSERT INTO web_user_two_factor (user_id, secret, source, recovery_hashes)
			SELECT id, ?, ?, ? FROM web_users WHERE id = ? AND created_at = ?`,
			next.Secret, next.Source, string(hashes), userID, formatWebUIStoreTime(next.accountCreated))
	default:
		result, err = s.store.db.Exec(`UPDATE web_user_two_factor SET recovery_hashes = ?, last_step = ? WHERE user_id = ? AND secret = ?`, string(hashes), next.LastStep, userID, previous.Secret)
	}
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count != 1 {
		return errTwoFactorChanged
	}
	return err
}

func (s *server) accountTwoFactorStatus(userID int64) (twoFactorStatus, error) {
	var source string
	if userID == 0 {
		record, err := s.readTwoFactor(0)
		if err != nil || record == nil {
			return twoFactorStatus{}, err
		}
		source = record.Source
	} else {
		err := s.store.db.QueryRow(`SELECT source FROM web_user_two_factor WHERE user_id = ?`, userID).Scan(&source)
		if errors.Is(err, sql.ErrNoRows) {
			return twoFactorStatus{}, nil
		}
		if err != nil {
			return twoFactorStatus{}, err
		}
	}
	return twoFactorStatus{Enabled: true, Source: source}, nil
}
