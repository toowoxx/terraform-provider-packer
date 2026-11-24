// SPDX-License-Identifier: MPL-2.0

// Package jwt stands in for github.com/golang-jwt/jwt v3 and forwards the part
// of its API that github.com/hashicorp/packer-plugin-azure uses to
// github.com/golang-jwt/jwt/v4. The v3 line has no release fixing
// GHSA-mh63-6h87-95cp, and the embedded Packer cannot be moved off it, so the
// root go.mod replaces v3 with this module and v4 does the work. Remove the
// replace and this module once `go list -deps ./...` in the root no longer
// lists github.com/golang-jwt/jwt.
package jwt

import jwtv4 "github.com/golang-jwt/jwt/v4"

type (
	Parser         = jwtv4.Parser
	StandardClaims = jwtv4.StandardClaims
	MapClaims      = jwtv4.MapClaims
	Token          = jwtv4.Token
)

var (
	SigningMethodRS256 = jwtv4.SigningMethodRS256
	SigningMethodES256 = jwtv4.SigningMethodES256
)

func New(method jwtv4.SigningMethod) *Token {
	return jwtv4.New(method)
}
