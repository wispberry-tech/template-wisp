// internal/compiler/bytecode.go
package compiler

// Opcode is a single VM instruction opcode.
type Opcode uint8

// Instruction is a fixed-width 8-byte VM instruction.
// Field layout: A(2) + B(2) + Op(1) + Flags(1) + _(2) = 8 bytes.
type Instruction struct {
	A     uint16  // primary operand (const index, name index, jump target, arg count)
	B     uint16  // secondary operand (argc for FILTER)
	Op    Opcode
	Flags uint8   // modifier bits (e.g. escape flag)
	_     [2]byte // reserved
}

const (
	OP_HALT       Opcode = iota
	OP_PUSH_CONST        // A = index into Consts
	OP_PUSH_NIL
	OP_LOAD              // A = index into Names — scope lookup
	OP_GET_ATTR          // A = index into Names — pop obj, push obj.Names[A]
	OP_GET_INDEX         // pop key, pop obj, push obj[key]
	OP_OUTPUT            // pop value, HTML-escape and write (unless SafeHTML)
	OP_OUTPUT_RAW        // pop value, write verbatim (no escaping)
	OP_ADD
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD
	OP_CONCAT   // ~ operator: pop b, pop a, push a+b as string
	OP_EQ
	OP_NEQ
	OP_LT
	OP_LTE
	OP_GT
	OP_GTE
	OP_AND
	OP_OR
	OP_NOT
	OP_NEGATE           // unary minus
	OP_JUMP             // A = target instruction index (unconditional)
	OP_JUMP_FALSE       // A = target; pop value, jump if falsy
	OP_FILTER           // A = name index, B = argc; pop argc args then value, push result

	// ─── Control flow opcodes (Plan 2) ────────────────────────────────────────
	OP_STORE_VAR     // A=name_idx; pop value, store to local scope (set)
	OP_PUSH_SCOPE    // push a new child scope (with)
	OP_POP_SCOPE     // pop to parent scope (endwith)
	OP_FOR_INIT      // A=fallthrough_ip; pop collection, push loop state; if empty jump to A
	OP_FOR_BIND_1    // A=var_name_idx; bind items[idx] to scope; bind "loop" map
	OP_FOR_BIND_KV   // A=key_idx B=val_idx; bind sorted key+val (map iteration) or index+val (list two-var)
	OP_FOR_STEP      // A=loop_top_ip; advance idx; if more jump to A; else pop loop state
	OP_CAPTURE_START // A=var_name_idx; redirect output to capture buffer
	OP_CAPTURE_END   // flush capture to scope[A]; restore output
	OP_CALL_RANGE    // A=argc; pop argc int args, push []Value list per range semantics

	// ─── Plan 4 opcodes ────────────────────────────────────────────────────────
	OP_MACRO_DEF       // A=name_idx B=macro_idx; store MacroDef as MacroVal in scope
	OP_MACRO_DEF_PUSH  // A=macro_idx; push MacroVal onto stack (for caller body)
	OP_CALL_MACRO_VAL  // A=posArgCount Flags=namedArgCount; pop namedArgs*2, posArgs, macroVal; push SafeHTML result
	OP_CALL_MACRO_CALL // like OP_CALL_MACRO_VAL but also pops caller body (MacroVal) beneath macro
	OP_CALL_CALLER     // call the __caller__ macro in current scope; push SafeHTML result
	OP_INCLUDE         // A=name_idx Flags: bit0=isolated; B=with_pair_count
	OP_RENDER          // A=name_idx B=with_pair_count; always isolated
	OP_IMPORT          // A=name_idx B=alias_idx
)

// MacroParam is a single parameter in a compiled macro.
type MacroParam struct {
	Name    string
	Default any // nil = required; string/int64/float64/bool = default constant
}

// MacroDef is a compiled macro: parameter list + body bytecode.
// Stored in Bytecode.Macros; referenced by index from OP_MACRO_DEF.
type MacroDef struct {
	Name   string
	Params []MacroParam
	Body   *Bytecode
}

// Bytecode is the compiled output for a single template.
// It is immutable after compilation and safe for concurrent use.
type Bytecode struct {
	Instrs []Instruction
	Consts []any      // constant pool: string | int64 | float64 | bool
	Names  []string   // name pool: variable names, attribute names, filter names
	Macros []MacroDef // compiled inline macros (referenced by OP_MACRO_DEF)
}
