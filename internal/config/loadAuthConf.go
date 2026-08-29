package config

import (
	"fmt"
	"time"

	"gopkg.in/ini.v1"
)

const authSection = "AUTH"

const (
	defaultAccessSecretKey       = ""
	defaultRefreshSecretKey      = ""
	defaultAccessTokenDuration   = 15 * time.Minute
	defaultRefreshTokenDuration  = 7 * 24 * time.Hour
	defaultCookieSecure          = false
	minJWTSecretLength           = 32
)

func loadAuthConfig(section *ini.Section) (AuthConfig, error) {
	accessSecretKey, err := ResolveString(authSection, "ACCESS_SECRET_KEY", configKey(section, "ACCESS_SECRET_KEY"), defaultAccessSecretKey)
	if err != nil {
		return AuthConfig{}, err
	}
	refreshSecretKey, err := ResolveString(authSection, "REFRESH_SECRET_KEY", configKey(section, "REFRESH_SECRET_KEY"), defaultRefreshSecretKey)
	if err != nil {
		return AuthConfig{}, err
	}
	accessTokenDuration, err := ResolveDuration(authSection, "ACCESS_TOKEN_DURATION", configKey(section, "ACCESS_TOKEN_DURATION"), defaultAccessTokenDuration)
	if err != nil {
		return AuthConfig{}, err
	}
	refreshTokenDuration, err := ResolveDuration(authSection, "REFRESH_TOKEN_DURATION", configKey(section, "REFRESH_TOKEN_DURATION"), defaultRefreshTokenDuration)
	if err != nil {
		return AuthConfig{}, err
	}
	cookieSecure, err := ResolveBool(authSection, "COOKIE_SECURE", configKey(section, "COOKIE_SECURE"), defaultCookieSecure)
	if err != nil {
		return AuthConfig{}, err
	}

	authConfig := AuthConfig{
		AccessSecretKey:      accessSecretKey,
		RefreshSecretKey:     refreshSecretKey,
		AccessTokenDuration:  accessTokenDuration,
		RefreshTokenDuration: refreshTokenDuration,
		CookieSecure:         cookieSecure,
	}
	if err := authConfig.validate(); err != nil {
		return AuthConfig{}, err
	}

	return authConfig, nil
}

func (c AuthConfig) validate() error {
	if len(c.AccessSecretKey) < minJWTSecretLength {
		return fmt.Errorf("access secret key must be at least %d characters (high-entropy random)", minJWTSecretLength)
	}
	if len(c.RefreshSecretKey) < minJWTSecretLength {
		return fmt.Errorf("refresh secret key must be at least %d characters (high-entropy random)", minJWTSecretLength)
	}
	if c.AccessSecretKey == c.RefreshSecretKey {
		return fmt.Errorf("access and refresh secret keys must be different")
	}
	if c.AccessTokenDuration <= 0 {
		return fmt.Errorf("access token duration must be greater than zero")
	}
	if c.RefreshTokenDuration <= 0 {
		return fmt.Errorf("refresh token duration must be greater than zero")
	}
	if c.RefreshTokenDuration <= c.AccessTokenDuration {
		return fmt.Errorf("refresh token duration must be greater than access token duration")
	}
	return nil
}
