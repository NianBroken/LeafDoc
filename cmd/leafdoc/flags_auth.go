package main

import (
	"time"

	"github.com/urfave/cli/v3"
)

// authOptions holds the options for authentication, tokens and the initial admin.
type authOptions struct {
	jwtSecret           string
	publicAccess        bool
	accessTokenTimeout  time.Duration
	refreshTokenTimeout time.Duration
	adminUsername       string
	adminEmail          string
	adminPassword       string
	editorLimit         int
}

// Flags declares them. Adding an option here is the whole change: the flag,
// its default, its environment variable and its help text are one literal,
// and --help is generated from it.
func (o *authOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "jwt-secret",
			Destination: &o.jwtSecret,
			Category:    catAuth,
			Usage:       "secret for signing auth tokens (JWT); required",
			Sources:     envVars("LEAFDOC_JWT_SECRET"),
			Config:      trimmed,
		},
		&cli.BoolFlag{
			Name:        "public-access",
			Destination: &o.publicAccess,
			Category:    catAuth,
			Usage:       "allow public access to the wiki with read access",
			Sources:     envBoolVars("LEAFDOC_PUBLIC_ACCESS"),
		},
		&cli.DurationFlag{
			Name:        "access-token-timeout",
			Destination: &o.accessTokenTimeout,
			Category:    catAuth,
			Usage:       "access token timeout duration (e.g. 24h, 15m)",
			Value:       15 * time.Minute,
			Sources:     envVars("LEAFDOC_ACCESS_TOKEN_TIMEOUT"),
			DefaultText: "15m",
		},
		&cli.DurationFlag{
			Name:        "refresh-token-timeout",
			Destination: &o.refreshTokenTimeout,
			Category:    catAuth,
			Usage:       "refresh token timeout duration (e.g. 168h)",
			Value:       7 * 24 * time.Hour,
			Sources:     envVars("LEAFDOC_REFRESH_TOKEN_TIMEOUT"),
			DefaultText: "168h",
		},
		&cli.StringFlag{
			Name:        "admin-username",
			Destination: &o.adminUsername,
			Category:    catAdmin,
			Usage:       "initial admin username, used only if no admin exists (default: admin)",
			Sources:     envVars("LEAFDOC_ADMIN_USERNAME"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "admin-email",
			Destination: &o.adminEmail,
			Category:    catAdmin,
			Usage:       "initial admin email, used only if no admin exists (default: admin@localhost)",
			Sources:     envVars("LEAFDOC_ADMIN_EMAIL"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "admin-password",
			Destination: &o.adminPassword,
			Category:    catAdmin,
			Usage:       "initial admin password, used only if no admin exists; min 8 characters",
			Sources:     envVars("LEAFDOC_ADMIN_PASSWORD"),
			Config:      trimmed,
		},
		&cli.IntFlag{
			Name:        "editor-limit",
			Destination: &o.editorLimit,
			Category:    catAuth,
			Usage:       "maximum admin+editor users allowed; 0 = unlimited",
			Sources:     envVars("LEAFDOC_EDITOR_LIMIT"),
		},
	}
}
