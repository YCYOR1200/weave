package auth

import (
	"context"

	"github.com/confio/weave"
	"github.com/confio/weave/x"
)

//------------------- Context --------
// Add context information specific to this package

type contextKey int // local to the auth module

const (
	contextKeySigners contextKey = iota
)

// withSigners is a private method, as only this module
// can add a signer
func withSigners(ctx weave.Context, signers []weave.Address) weave.Context {
	return context.WithValue(ctx, contextKeySigners, signers)
}

// GetSigners returns who signed the current Context.
// May be empty
func GetSigners(ctx weave.Context) []weave.Address {
	// (val, ok) form to return nil instead of panic if unset
	val, _ := ctx.Value(contextKeySigners).([]weave.Address)
	// if we were paranoid about our own code, we would deep-copy
	// the signers here
	return val
}

// Note that we expect this to be exported
var _ x.AuthFunc = GetSigners
