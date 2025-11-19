package expr

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"floe/memory"
)

// EvaluateBool evaluates a boolean expression string against the memory.
// It supports basic operators: ==, !=, >, <, >=, <=, &&, ||.
// Variables in the expression (e.g. ${var}) are substituted before evaluation.
func EvaluateBool(exprStr string, mem *memory.Memory) (bool, error) {
	// 1. Substitute variables
	// We use memory.ResolveInterpolation which does string replacement.
	// Note: Users must quote string variables in their YAML if they want them treated as strings.
	// e.g. "${var} == 'value'" -> "actual_value == 'value'" (if var is identifier-like)
	// or "'${var}' == 'value'" -> "'actual_value' == 'value'"
	resolvedExpr := mem.ResolveInterpolation(exprStr)

	// 2. Parse expression
	expr, err := parser.ParseExpr(resolvedExpr)
	if err != nil {
		return false, fmt.Errorf("failed to parse expression '%s' (resolved: '%s'): %w", exprStr, resolvedExpr, err)
	}

	// 3. Evaluate AST
	val, err := eval(expr)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression '%s': %w", resolvedExpr, err)
	}

	boolVal, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("expression '%s' did not evaluate to a boolean, got %T (%v)", resolvedExpr, val, val)
	}

	return boolVal, nil
}

// EvaluateString evaluates an expression that results in a string.
// This is primarily for dynamic 'next' routing where the expression is just a variable or a ternary (if we supported it).
// For now, it mainly handles variable substitution and simple string literals.
// If the expression is complex, we might need to evaluate it.
// But for v0.4, 'next' expression is usually just "${some_var}".
// If it's a complex expression like "cond ? 'a' : 'b'", we need full eval.
// Let's support full eval and expect a string result.
func EvaluateString(exprStr string, mem *memory.Memory) (string, error) {
	resolvedExpr := mem.ResolveInterpolation(exprStr)

	// If it looks like a simple string (no operators), just return it.
	// But "true ? 'a' : 'b'" needs parsing.
	// For now, let's try to parse. If it fails, maybe it's just a raw string?
	// But EvaluateString is called when we expect an expression.

	expr, err := parser.ParseExpr(resolvedExpr)
	if err != nil {
		// If it fails to parse, it might be a simple string that happens to have special chars?
		// But we assume expressions are valid Go expressions.
		// If the user provided a raw string "step_next", ResolveInterpolation returns "step_next".
		// parser.ParseExpr("step_next") parses as *ast.Ident.
		// eval(Ident) returns... error if unknown?
		// Let's handle Ident as string if it's not true/false.
		return resolvedExpr, nil
	}

	val, err := eval(expr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", val), nil
}

func eval(node ast.Node) (interface{}, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return evalBasicLit(n)
	case *ast.Ident:
		return evalIdent(n)
	case *ast.BinaryExpr:
		return evalBinaryExpr(n)
	case *ast.ParenExpr:
		return eval(n.X)
	case *ast.UnaryExpr:
		return evalUnaryExpr(n)
	default:
		return nil, fmt.Errorf("unsupported expression node type: %T", n)
	}
}

func evalBasicLit(n *ast.BasicLit) (interface{}, error) {
	switch n.Kind {
	case token.INT:
		return strconv.Atoi(n.Value)
	case token.STRING:
		return strconv.Unquote(n.Value)
	case token.CHAR:
		return strconv.Unquote(n.Value)
	default:
		return nil, fmt.Errorf("unsupported literal type: %v", n.Kind)
	}
}

func evalIdent(n *ast.Ident) (interface{}, error) {
	switch n.Name {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "nil":
		return nil, nil
	default:
		// Treat unknown identifiers as strings to allow unquoted strings like 'step_name'
		// This is a bit loose but helpful for "next: step_name" if parsed as expression.
		return n.Name, nil
	}
}

func evalUnaryExpr(n *ast.UnaryExpr) (interface{}, error) {
	val, err := eval(n.X)
	if err != nil {
		return nil, err
	}

	switch n.Op {
	case token.NOT:
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid type for ! operator: %T", val)
		}
		return !b, nil
	default:
		return nil, fmt.Errorf("unsupported unary operator: %v", n.Op)
	}
}

func evalBinaryExpr(n *ast.BinaryExpr) (interface{}, error) {
	left, err := eval(n.X)
	if err != nil {
		return nil, err
	}
	right, err := eval(n.Y)
	if err != nil {
		return nil, err
	}

	switch n.Op {
	case token.EQL: // ==
		return left == right, nil
	case token.NEQ: // !=
		return left != right, nil
	case token.LAND: // &&
		l, ok1 := left.(bool)
		r, ok2 := right.(bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("&& requires booleans")
		}
		return l && r, nil
	case token.LOR: // ||
		l, ok1 := left.(bool)
		r, ok2 := right.(bool)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("|| requires booleans")
		}
		return l || r, nil
	case token.GTR, token.LSS, token.GEQ, token.LEQ:
		return evalCompare(n.Op, left, right)
	default:
		return nil, fmt.Errorf("unsupported binary operator: %v", n.Op)
	}
}

func evalCompare(op token.Token, left, right interface{}) (bool, error) {
	// Handle numeric comparison
	lInt, lOk := toInt(left)
	rInt, rOk := toInt(right)

	if lOk && rOk {
		switch op {
		case token.GTR:
			return lInt > rInt, nil
		case token.LSS:
			return lInt < rInt, nil
		case token.GEQ:
			return lInt >= rInt, nil
		case token.LEQ:
			return lInt <= rInt, nil
		}
	}

	// Handle string comparison (only for <, >, etc? usually only ==/!= for strings in simple DSLs, but Go supports string comparison)
	lStr, lStrOk := left.(string)
	rStr, rStrOk := right.(string)
	if lStrOk && rStrOk {
		switch op {
		case token.GTR:
			return lStr > rStr, nil
		case token.LSS:
			return lStr < rStr, nil
		case token.GEQ:
			return lStr >= rStr, nil
		case token.LEQ:
			return lStr <= rStr, nil
		}
	}

	return false, fmt.Errorf("invalid types for comparison: %T and %T", left, right)
}

func toInt(v interface{}) (int, bool) {
	if i, ok := v.(int); ok {
		return i, true
	}
	return 0, false
}
