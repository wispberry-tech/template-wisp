// internal/compiler/compiler.go
package compiler

import (
	"fmt"

	"grove/internal/ast"
)

// Compile walks prog and emits Bytecode.
func Compile(prog *ast.Program) (*Bytecode, error) {
	c := &cmp{nameIdx: make(map[string]int)}
	if err := c.compileProgram(prog); err != nil {
		return nil, err
	}
	c.emit(OP_HALT, 0, 0, 0)
	return &Bytecode{Instrs: c.instrs, Consts: c.consts, Names: c.names}, nil
}

type cmp struct {
	instrs  []Instruction
	consts  []any
	names   []string
	nameIdx map[string]int
}

func (c *cmp) compileProgram(prog *ast.Program) error {
	for _, node := range prog.Body {
		if err := c.compileNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (c *cmp) compileBody(nodes []ast.Node) error {
	for _, node := range nodes {
		if err := c.compileNode(node); err != nil {
			return err
		}
	}
	return nil
}

func (c *cmp) compileNode(node ast.Node) error {
	switch n := node.(type) {
	case *ast.TextNode:
		c.emitPushConst(n.Value)
		c.emit(OP_OUTPUT_RAW, 0, 0, 0)

	case *ast.RawNode:
		c.emitPushConst(n.Value)
		c.emit(OP_OUTPUT_RAW, 0, 0, 0)

	case *ast.OutputNode:
		if err := c.compileExpr(n.Expr); err != nil {
			return err
		}
		c.emit(OP_OUTPUT, 0, 0, 0)

	case *ast.TagNode:
		// Unimplemented tags are no-ops
		return nil

	case *ast.IfNode:
		return c.compileIf(n)

	case *ast.UnlessNode:
		return c.compileUnless(n)

	case *ast.ForNode:
		return c.compileFor(n)

	case *ast.SetNode:
		if err := c.compileExpr(n.Expr); err != nil {
			return err
		}
		c.emit(OP_STORE_VAR, uint16(c.addName(n.Name)), 0, 0)

	case *ast.WithNode:
		c.emit(OP_PUSH_SCOPE, 0, 0, 0)
		if err := c.compileBody(n.Body); err != nil {
			return err
		}
		c.emit(OP_POP_SCOPE, 0, 0, 0)

	case *ast.CaptureNode:
		c.emit(OP_CAPTURE_START, uint16(c.addName(n.Name)), 0, 0)
		if err := c.compileBody(n.Body); err != nil {
			return err
		}
		c.emit(OP_CAPTURE_END, uint16(c.addName(n.Name)), 0, 0)

	default:
		return fmt.Errorf("compiler: unknown node type %T", node)
	}
	return nil
}

// ─── {% if %} compiler ────────────────────────────────────────────────────────

func (c *cmp) compileIf(n *ast.IfNode) error {
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)

	if err := c.compileBody(n.Body); err != nil {
		return err
	}

	var endJumps []int
	endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
	c.instrs[jfIdx].A = uint16(len(c.instrs))

	for _, elif := range n.Elifs {
		if err := c.compileExpr(elif.Condition); err != nil {
			return err
		}
		elifJfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileBody(elif.Body); err != nil {
			return err
		}
		endJumps = append(endJumps, c.emitPlaceholder(OP_JUMP))
		c.instrs[elifJfIdx].A = uint16(len(c.instrs))
	}

	if len(n.Else) > 0 {
		if err := c.compileBody(n.Else); err != nil {
			return err
		}
	}

	end := uint16(len(c.instrs))
	for _, jIdx := range endJumps {
		c.instrs[jIdx].A = end
	}

	return nil
}

// ─── {% unless %} compiler ────────────────────────────────────────────────────

func (c *cmp) compileUnless(n *ast.UnlessNode) error {
	if err := c.compileExpr(n.Condition); err != nil {
		return err
	}
	c.emit(OP_NOT, 0, 0, 0)
	jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
	if err := c.compileBody(n.Body); err != nil {
		return err
	}
	c.instrs[jfIdx].A = uint16(len(c.instrs))
	return nil
}

// ─── {% for %} compiler ───────────────────────────────────────────────────────

func (c *cmp) compileFor(n *ast.ForNode) error {
	if err := c.compileExpr(n.Iterable); err != nil {
		return err
	}

	forInitIdx := c.emitPlaceholder(OP_FOR_INIT)

	loopTop := uint16(len(c.instrs))
	if n.Var2 == "" {
		c.emit(OP_FOR_BIND_1, uint16(c.addName(n.Var1)), 0, 0)
	} else {
		c.emit(OP_FOR_BIND_KV, uint16(c.addName(n.Var1)), uint16(c.addName(n.Var2)), 0)
	}

	if err := c.compileBody(n.Body); err != nil {
		return err
	}

	c.emit(OP_FOR_STEP, loopTop, 0, 0)

	if len(n.Empty) > 0 {
		jumpPastEmptyIdx := c.emitPlaceholder(OP_JUMP)
		c.instrs[forInitIdx].A = uint16(len(c.instrs))
		if err := c.compileBody(n.Empty); err != nil {
			return err
		}
		c.instrs[jumpPastEmptyIdx].A = uint16(len(c.instrs))
	} else {
		c.instrs[forInitIdx].A = uint16(len(c.instrs))
	}

	return nil
}

// ─── Expression compiler ──────────────────────────────────────────────────────

func (c *cmp) compileExpr(node ast.Node) error {
	switch n := node.(type) {
	case *ast.NilLiteral:
		c.emit(OP_PUSH_NIL, 0, 0, 0)

	case *ast.BoolLiteral:
		c.emitPushConst(n.Value)

	case *ast.IntLiteral:
		c.emitPushConst(n.Value)

	case *ast.FloatLiteral:
		c.emitPushConst(n.Value)

	case *ast.StringLiteral:
		c.emitPushConst(n.Value)

	case *ast.Identifier:
		c.emit(OP_LOAD, uint16(c.addName(n.Name)), 0, 0)

	case *ast.AttributeAccess:
		if err := c.compileExpr(n.Object); err != nil {
			return err
		}
		c.emit(OP_GET_ATTR, uint16(c.addName(n.Key)), 0, 0)

	case *ast.IndexAccess:
		if err := c.compileExpr(n.Object); err != nil {
			return err
		}
		if err := c.compileExpr(n.Key); err != nil {
			return err
		}
		c.emit(OP_GET_INDEX, 0, 0, 0)

	case *ast.BinaryExpr:
		if err := c.compileExpr(n.Left); err != nil {
			return err
		}
		if err := c.compileExpr(n.Right); err != nil {
			return err
		}
		switch n.Op {
		case "+":
			c.emit(OP_ADD, 0, 0, 0)
		case "-":
			c.emit(OP_SUB, 0, 0, 0)
		case "*":
			c.emit(OP_MUL, 0, 0, 0)
		case "/":
			c.emit(OP_DIV, 0, 0, 0)
		case "%":
			c.emit(OP_MOD, 0, 0, 0)
		case "~":
			c.emit(OP_CONCAT, 0, 0, 0)
		case "==":
			c.emit(OP_EQ, 0, 0, 0)
		case "!=":
			c.emit(OP_NEQ, 0, 0, 0)
		case "<":
			c.emit(OP_LT, 0, 0, 0)
		case "<=":
			c.emit(OP_LTE, 0, 0, 0)
		case ">":
			c.emit(OP_GT, 0, 0, 0)
		case ">=":
			c.emit(OP_GTE, 0, 0, 0)
		case "and":
			c.emit(OP_AND, 0, 0, 0)
		case "or":
			c.emit(OP_OR, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown binary op %q", n.Op)
		}

	case *ast.UnaryExpr:
		if err := c.compileExpr(n.Operand); err != nil {
			return err
		}
		switch n.Op {
		case "not":
			c.emit(OP_NOT, 0, 0, 0)
		case "-":
			c.emit(OP_NEGATE, 0, 0, 0)
		default:
			return fmt.Errorf("compiler: unknown unary op %q", n.Op)
		}

	case *ast.TernaryExpr:
		if err := c.compileExpr(n.Condition); err != nil {
			return err
		}
		jfIdx := c.emitPlaceholder(OP_JUMP_FALSE)
		if err := c.compileExpr(n.Consequence); err != nil {
			return err
		}
		jIdx := c.emitPlaceholder(OP_JUMP)
		c.instrs[jfIdx].A = uint16(len(c.instrs))
		if err := c.compileExpr(n.Alternative); err != nil {
			return err
		}
		c.instrs[jIdx].A = uint16(len(c.instrs))

	case *ast.FilterExpr:
		if err := c.compileExpr(n.Value); err != nil {
			return err
		}
		for _, arg := range n.Args {
			if err := c.compileExpr(arg); err != nil {
				return err
			}
		}
		c.emit(OP_FILTER, uint16(c.addName(n.Filter)), uint16(len(n.Args)), 0)

	case *ast.FuncCallNode:
		switch n.Name {
		case "range":
			for _, arg := range n.Args {
				if err := c.compileExpr(arg); err != nil {
					return err
				}
			}
			c.emit(OP_CALL_RANGE, uint16(len(n.Args)), 0, 0)
		default:
			return fmt.Errorf("compiler: unknown function %q", n.Name)
		}

	default:
		return fmt.Errorf("compiler: unknown expr type %T", node)
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (c *cmp) emit(op Opcode, a, b uint16, flags uint8) {
	c.instrs = append(c.instrs, Instruction{Op: op, A: a, B: b, Flags: flags})
}

// emitPlaceholder emits an instruction with A=0 and returns its index for back-patching.
func (c *cmp) emitPlaceholder(op Opcode) int {
	idx := len(c.instrs)
	c.emit(op, 0, 0, 0)
	return idx
}

func (c *cmp) emitPushConst(v any) {
	idx := len(c.consts)
	c.consts = append(c.consts, v)
	c.emit(OP_PUSH_CONST, uint16(idx), 0, 0)
}

func (c *cmp) addName(name string) int {
	if idx, ok := c.nameIdx[name]; ok {
		return idx
	}
	idx := len(c.names)
	c.names = append(c.names, name)
	c.nameIdx[name] = idx
	return idx
}
