//go:build !groveprofile

package vm

import "github.com/wispberry-tech/grove/internal/compiler"

// OpcodeStats is a no-op placeholder when profiling is disabled.
type OpcodeStats struct{}

func ResetOpcodeStats()            {}
func GetOpcodeStats() OpcodeStats  { return OpcodeStats{} }
func (OpcodeStats) String() string { return "" }

// CatNames is empty when profiling is disabled.
var CatNames [0]string

const CatCount = 0

type profileState struct{}

func profileInit() profileState                          { return profileState{} }
func profileRecord(_ *profileState, _ compiler.Opcode)   {}
func profileFlush(_ *profileState)                       {}
