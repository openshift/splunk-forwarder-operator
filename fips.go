//go:build fips_enabled
// +build fips_enabled

// BOILERPLATE GENERATED -- DO NOT EDIT
// Run 'make ensure-fips' to regenerate

package main

import (
	_ "crypto/tls/fipsonly"
	"fmt"
	"os"
)

func init() {
	_, _ = fmt.Fprintln(os.Stderr, "***** Starting with FIPS crypto enabled *****")
}
