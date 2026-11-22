package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

func validDiscordToken(token string) bool {
	if len(token) == 0 || len(token) > 2048 {
		return false
	}
	for _, c := range token {
		// Credentials are opaque printable ASCII, without any segmentation rule.
		if c < '!' || c > '~' {
			return false
		}
	}
	return true
}

func validDiscordSnowflake(id string) bool {
	if len(id) == 0 || len(id) > 20 || id[0] == '0' {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(id, 10, 64)
	return err == nil && n != 0
}

var discordHTTP = &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// Only these two fixed GET endpoints are available in Stage 1A. Never return
// transport diagnostics or Discord response bodies, which may contain secrets.
func discordRequest(ctx context.Context, token, endpoint string, output any) error {
	if endpoint != "/users/@me" && endpoint != "/applications/@me" {
		return errors.New("Discord request denied")
	}
	if !validDiscordToken(token) {
		return errors.New("Discord credential invalid")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/v10"+endpoint, nil)
	if err != nil {
		return errors.New("Discord request unavailable")
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DiscordBot (https://github.com/home-server-project/justvoxel, 1.0)")
	response, err := discordHTTP.Do(req)
	if err != nil {
		return errors.New("Discord connection unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("Discord verification failed")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return errors.New("Discord response unavailable")
	}
	if json.Unmarshal(raw, output) != nil {
		return errors.New("Discord response invalid")
	}
	return nil
}
