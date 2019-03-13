package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"sync"
)

const (
	VAR_VARIABLE byte = iota
	VAR_NUMBER
	VAR_STRING
	VAR_CHAR
	VAR_FUNCTION
	VAR_NFUNCTION
	VAR_ARRAY
	VAR_OBJECT
	VAR_RETURN
	VAR_BREAK
	VAR_CONTINUE
)

type Variable struct {
	Type   byte
	Value  interface{}
	Parent *Variable
}

type VariableChar struct {
	Str *Variable
	I   int
}

const (
	STACK_POINTER int = iota
	STACK_MAP
)

type Stack struct {
	Type    int
	Pointer int
	Map     map[string]*Variable
}

var global_stack = make(map[string]*Variable)
var global_stack_mu = &sync.RWMutex{}

func Error(t Token, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%d:%d: ", t.Line, t.Col)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func MakeVariable(t byte, v interface{}) Variable {
	return Variable{Type: t, Value: v}
}

func StackPush(name string, thread *[]Stack, index int, variable Variable) {
	if index < 0 {
		global_stack_mu.Lock()
		global_stack[name] = &variable
		global_stack_mu.Unlock()
	} else {
		(*thread)[index].Map[name] = &variable
	}
}

func StackGet(name string, thread []Stack, i int) *Variable {
	for ; i >= 0; i-- {
		if s := thread[i]; s.Type == STACK_POINTER {
			i = s.Pointer + 1
		} else if v, ok := s.Map[name]; ok {
			return v
		}
	}

	global_stack_mu.RLock()
	ret := global_stack[name]
	global_stack_mu.RUnlock()
	return ret
}

func StackAdd(thread *[]Stack, last int) int {
	t := *thread
	t = append(t, Stack{Type: STACK_POINTER, Pointer: last}, Stack{Type: STACK_MAP, Map: make(map[string]*Variable)})
	*thread = t
	return len(t) - 1
}

func StackRemove(thread *[]Stack) {
	t := *thread
	*thread = t[0 : len(t)-2]
}

func ToString(variable Variable) string {
	switch variable.Type {
	case VAR_VARIABLE:
		return ToString(*variable.Value.(*Variable))
	case VAR_NUMBER:
		return strconv.FormatFloat(variable.Value.(float64), 'f', -1, 64)
		// return fmt.Sprint(variable.Value.(float64))
	case VAR_STRING:
		return variable.Value.(string)
	case VAR_FUNCTION:
		return fmt.Sprint(&variable.Value)
	case VAR_ARRAY:
		{
			s := "["
			for i, e := range *variable.Value.(*[]Variable) {
				if i > 0 {
					s += ","
				}

				s += ToString(e)
			}
			return s + "]"
		}
	case VAR_OBJECT:
		{
			s := "{"
			f := false
			for k, v := range *variable.Value.(*map[string]*Variable) {
				if f {
					s += ","
				} else {
					f = true
				}

				s += k + ":" + ToString(*v)
			}
			return s + "}"
		}
	}

	return fmt.Sprint(variable.Value)
}

/* func ToString(variable Variable) string {
	switch variable.Type {
	case VAR_VARIABLE:
		return ToString(*variable.Value.(*Variable))
	case VAR_NUMBER:
		return fmt.Sprint(variable.Value.(float64))
	case VAR_STRING:
		return variable.Value.(string)
	case VAR_FUNCTION:
		return fmt.Sprint("%p", &variable.Value)
	case VAR_ARRAY:
		{
			s := "["
			for i, e := range *variable.Value.(*[]Variable) {
				if i > 0 {
					s += ", "
				}
				s += ToString(e)
			}
			return s + "]"
		}
	case VAR_OBJECT:
		{
			s := "{\n"
			i := 0
			for k, v := range *variable.Value.(*map[string]*Variable) {
				if i > 0 {
					s += ",\n"
				}
				i++
				s += "  " + k + ": " + strings.Replace(ToString(*v), "\n", "\n  ", -1)
			}
			return s + "\n}"
		}
	}

	return fmt.Sprint(variable.Value)
} */

func ReduceVariable(variable Variable) Variable {
	if variable.Type == VAR_VARIABLE {
		if v := variable.Value.(*Variable); v != nil {
			return ReduceVariable(*variable.Value.(*Variable))
		} else {
			return MakeVariable(VAR_NUMBER, float64(0))
		}
	} else if variable.Type == VAR_RETURN {
		return ReduceVariable(variable.Value.(Variable))
	} else if variable.Type == VAR_CHAR {
		c := variable.Value.(VariableChar)
		variable = MakeVariable(VAR_STRING, string([]rune((*c.Str).Value.(string))[c.I]))
	}

	return variable
}

func Eval(tree Tree, thread *[]Stack, stack int) Variable {
	switch tree.T.Type {
	case TOKEN_BLOCK:
		{
			result := MakeVariable(VAR_NUMBER, float64(0))

			for _, e := range tree.C {
				result = Eval(e, thread, stack)
				if result.Type == VAR_RETURN || result.Type == VAR_BREAK || result.Type == VAR_CONTINUE {
					break
				}
			}

			return result
		}
	case TOKEN_FUNC:
		{
			v := MakeVariable(VAR_FUNCTION, tree.C)
			if tree.T.Value != nil {
				StackPush(tree.T.Value.(string), thread, stack, v)
			}
			return v
		}
	case TOKEN_CALL:
		{
			v := ReduceVariable(Eval(tree.C[0], thread, stack))
			if v.Type == VAR_FUNCTION {
				f := v.Value.([]Tree)

				old_stack := stack
				stack = StackAdd(thread, -1)

				var args []Variable
				for i := 1; i < len(tree.C); i++ {
					args = append(args, ReduceVariable(Eval(tree.C[i], thread, old_stack)))
				}

				StackPush("arguments", thread, stack, MakeVariable(VAR_ARRAY, &args))
				if v.Parent != nil {
					StackPush("this", thread, stack, *v.Parent)
				}

				for i, e := range f[0].C {
					if i < len(args) {
						StackPush(e.T.Value.(string), thread, stack, args[i])
					} else {
						StackPush(e.T.Value.(string), thread, stack, MakeVariable(VAR_NUMBER, float64(0)))
					}
				}

				ret := MakeVariable(VAR_NUMBER, float64(0))

				for _, e := range f[1].C {
					if v := Eval(e, thread, stack); v.Type == VAR_RETURN {
						ret = v.Value.(Variable)
						break
					}
				}

				StackRemove(thread)
				return ret
			} else if v.Type == VAR_NFUNCTION {
				var args []Variable
				for i := 1; i < len(tree.C); i++ {
					args = append(args, ReduceVariable(Eval(tree.C[i], thread, stack)))
				}

				return v.Value.(func(*Variable, []Variable) Variable)(v.Parent, args)
			}

			Error(tree.T, `call on a non-function variable`)
		}
	case TOKEN_RETURN:
		{
			if len(tree.C) > 0 {
				return MakeVariable(VAR_RETURN, ReduceVariable(Eval(tree.C[0], thread, stack)))
			}

			return MakeVariable(VAR_RETURN, MakeVariable(VAR_NUMBER, float64(0)))
		}
	case TOKEN_BREAK:
		{
			return MakeVariable(VAR_BREAK, 0)
		}
	case TOKEN_CONTINUE:
		{
			return MakeVariable(VAR_CONTINUE, 0)
		}
	case TOKEN_IF:
		{
			if v := ReduceVariable(Eval(tree.C[0], thread, stack)); v.Type != VAR_NUMBER || v.Value.(float64) != 0 {
				s := StackAdd(thread, stack)
				ret := Eval(tree.C[1], thread, s)
				StackRemove(thread)
				return ret
			} else if len(tree.C) > 2 {
				if tree.C[2].T.Type == TOKEN_IF {
					return Eval(tree.C[2], thread, stack)
				} else {
					s := StackAdd(thread, stack)
					ret := Eval(tree.C[2], thread, s)
					StackRemove(thread)
					return ret
				}
			}
		}
	case TOKEN_FOR:
		{
			stack = StackAdd(thread, stack)
			Eval(tree.C[0], thread, stack)

			ret := MakeVariable(VAR_NUMBER, 0)
			for {
				if !IsNoTree(tree.C[1]) {
					if v := Eval(tree.C[1], thread, stack); v.Type == VAR_NUMBER && v.Value.(float64) == 0 {
						break
					}
				}

				ret = Eval(tree.C[3], thread, stack)
				if ret.Type == VAR_RETURN {
					break
				} else if ret.Type == VAR_BREAK {
					ret.Type = VAR_NUMBER
					break
				} else if ret.Type == VAR_CONTINUE {
					ret.Type = VAR_NUMBER
				}

				Eval(tree.C[2], thread, stack)
			}

			StackRemove(thread)
			return ret
		}
	case TOKEN_WHILE:
		{
			old_stack := stack
			stack = StackAdd(thread, stack)

			ret := MakeVariable(VAR_NUMBER, 0)
			for {
				if v := Eval(tree.C[0], thread, old_stack); v.Type == VAR_NUMBER && v.Value.(float64) == 0 {
					break
				}

				ret = Eval(tree.C[1], thread, stack)
				if ret.Type == VAR_RETURN {
					break
				} else if ret.Type == VAR_BREAK {
					ret.Type = VAR_NUMBER
					break
				} else if ret.Type == VAR_CONTINUE {
					ret.Type = VAR_NUMBER
				}
			}

			StackRemove(thread)
			return ret
		}
	case TOKEN_QUESTION:
		{
			e := ReduceVariable(Eval(tree.C[0], thread, stack))
			if e.Type != VAR_NUMBER || e.Value.(float64) != 0 {
				return ReduceVariable(Eval(tree.C[1], thread, stack))
			}

			return ReduceVariable(Eval(tree.C[2], thread, stack))
		}
	case TOKEN_DECLARATION:
		{
			v1 := tree.C[0].T
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))

			if v1.Type == TOKEN_WORD {
				StackPush(v1.Value.(string), thread, stack, v2)
				return v2
			}

			Error(tree.T, `invalid left-hand side in declaration`)
		}
	case TOKEN_EQUAL:
		{
			v1 := Eval(tree.C[0], thread, stack)
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))

			if v1.Type == VAR_VARIABLE {
				if v := v1.Value.(*Variable); v != nil {
					v2.Parent = v.Parent
					*v = v2
					return v2
				} else {
					Error(tree.T, `variable on left-hand side in assignment is undefined`)
					break
				}
			} else if v1.Type == VAR_CHAR {
				c := v1.Value.(VariableChar)
				s := (*c.Str).Value.(string)
				n := MakeVariable(VAR_STRING, s[0:c.I]+ToString(v2)+s[c.I+1:])
				*c.Str = n
				return n
			}

			Error(tree.T, `invalid left-hand side in assignment`)
		}
	case TOKEN_WORD:
		return MakeVariable(VAR_VARIABLE, StackGet(tree.T.Value.(string), *thread, stack))
	case TOKEN_NUMBER:
		return MakeVariable(VAR_NUMBER, tree.T.Value.(float64))
	case TOKEN_STRING:
		return MakeVariable(VAR_STRING, tree.T.Value.(string))
	case TOKEN_OPEN_BRACKET:
		{
			var arr []Variable

			for _, e := range tree.C {
				arr = append(arr, ReduceVariable(Eval(e, thread, stack)))
			}

			return MakeVariable(VAR_ARRAY, &arr)
		}
	case TOKEN_OPEN_BRACE:
		{
			obj := make(map[string]*Variable)
			parent := MakeVariable(VAR_OBJECT, &obj)

			for _, e := range tree.C {
				v := ReduceVariable(Eval(e.C[0], thread, stack))
				v.Parent = &parent
				obj[e.T.Value.(string)] = &v
			}

			return parent
		}
	case TOKEN_INDEX:
		{
			o := Eval(tree.C[0], thread, stack)
			v := ReduceVariable(o)
			if v.Type == VAR_ARRAY {
				i := ReduceVariable(Eval(tree.C[1], thread, stack))
				if i.Type == VAR_NUMBER {
					index := int(i.Value.(float64))
					if arr := *v.Value.(*[]Variable); index >= 0 && index < len(arr) {
						return MakeVariable(VAR_VARIABLE, &arr[index])
					} else {
						Error(tree.C[1].T, `index is out of range`)
						break
					}
				} else {
					Error(tree.C[1].T, `index is not a number`)
					break
				}
			} else if v.Type == VAR_OBJECT {
				i := ReduceVariable(Eval(tree.C[1], thread, stack))
				if i.Type == VAR_STRING {
					key := i.Value.(string)
					m := *v.Value.(*map[string]*Variable)
					if e, ok := m[key]; ok {
						return MakeVariable(VAR_VARIABLE, e)
					} else {
						n := MakeVariable(VAR_NUMBER, float64(0))
						n.Parent = &v
						m[key] = &n
						return MakeVariable(VAR_VARIABLE, &n)
					}
				} else {
					Error(tree.T, `key is not a string`)
					break
				}
			} else if v.Type == VAR_STRING {
				i := ReduceVariable(Eval(tree.C[1], thread, stack))
				if i.Type == VAR_NUMBER {
					index := int(i.Value.(float64))
					if s := []rune(v.Value.(string)); index >= 0 && index < len(s) {
						return MakeVariable(VAR_CHAR, VariableChar{Str: o.Value.(*Variable), I: index})
					}
				}
			}

			Error(tree.T, `cannot index on non-array, non-object, or non-string`)
		}
	case TOKEN_PERIOD:
		{
			v := ReduceVariable(Eval(tree.C[0], thread, stack))
			if v.Type == VAR_OBJECT {
				key := tree.C[1].T.Value.(string)
				m := *v.Value.(*map[string]*Variable)
				if e, ok := m[key]; ok {
					return MakeVariable(VAR_VARIABLE, e)
				} else {
					n := MakeVariable(VAR_NUMBER, float64(0))
					n.Parent = &v
					m[key] = &n
					return MakeVariable(VAR_VARIABLE, &n)
				}
			}

			Error(tree.T, `cannot use accessor on non-object`)
		}
	case TOKEN_SIGN_PLUS:
		{
			if v := ReduceVariable(Eval(tree.C[0], thread, stack)); v.Type == VAR_NUMBER {
				return v
			}
		}
	case TOKEN_SIGN_MINUS:
		{
			if v := ReduceVariable(Eval(tree.C[0], thread, stack)); v.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, -v.Value.(float64))
			}
		}
	case TOKEN_NOT:
		{
			if v := ReduceVariable(Eval(tree.C[0], thread, stack)); v.Type == VAR_NUMBER && v.Value.(float64) == 0 {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_BIT_NOT:
		{
			if v := ReduceVariable(Eval(tree.C[0], thread, stack)); v.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, float64(^int(v.Value.(float64))))
			}
		}
	case TOKEN_PRE_INC:
		{
			if v1 := Eval(tree.C[0], thread, stack); v1.Type == VAR_VARIABLE {
				if v := v1.Value.(*Variable); v != nil && v.Type == VAR_NUMBER {
					r := v.Value.(float64) + 1
					v.Value = r
					return MakeVariable(VAR_NUMBER, r)
				}
			}

			Error(tree.T, `bad increment on non-variable`)
		}
	case TOKEN_POST_INC:
		{
			if v1 := Eval(tree.C[0], thread, stack); v1.Type == VAR_VARIABLE {
				if v := v1.Value.(*Variable); v != nil && v.Type == VAR_NUMBER {
					r := v.Value.(float64)
					v.Value = r + 1
					return MakeVariable(VAR_NUMBER, r)
				}
			}

			Error(tree.T, `bad increment on non-variable`)
		}
	case TOKEN_PRE_DEC:
		{
			if v1 := Eval(tree.C[0], thread, stack); v1.Type == VAR_VARIABLE {
				if v := v1.Value.(*Variable); v != nil && v.Type == VAR_NUMBER {
					r := v.Value.(float64) - 1
					v.Value = r
					return MakeVariable(VAR_NUMBER, r)
				}
			}

			Error(tree.T, `bad decrement on non-variable`)
		}
	case TOKEN_POST_DEC:
		{
			if v1 := Eval(tree.C[0], thread, stack); v1.Type == VAR_VARIABLE {
				if v := v1.Value.(*Variable); v != nil && v.Type == VAR_NUMBER {
					r := v.Value.(float64)
					v.Value = r - 1
					return MakeVariable(VAR_NUMBER, r)
				}
			}

			Error(tree.T, `bad decrement on non-variable`)
		}
	case TOKEN_PLUS:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))

			if v1.Type == VAR_STRING {
				return MakeVariable(VAR_STRING, v1.Value.(string)+ToString(v2))
			} else if v2.Type == VAR_STRING {
				return MakeVariable(VAR_STRING, ToString(v1)+v2.Value.(string))
			} else if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, v1.Value.(float64)+v2.Value.(float64))
			}
		}
	case TOKEN_MINUS:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, v1.Value.(float64)-v2.Value.(float64))
			}
		}
	case TOKEN_MULTIPLY:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, v1.Value.(float64)*v2.Value.(float64))
			}
		}
	case TOKEN_DIVIDE:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, v1.Value.(float64)/v2.Value.(float64))
			}
		}
	case TOKEN_MOD:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, math.Mod(v1.Value.(float64), v2.Value.(float64)))
			}
		}
	case TOKEN_EQUAL_EQUAL:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == v2.Type && v1.Value == v2.Value {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_NOT_EQUAL:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type != v2.Type || v1.Value != v2.Value {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_LESS:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER && v1.Value.(float64) < v2.Value.(float64) {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_GREATER:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER && v1.Value.(float64) > v2.Value.(float64) {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_LESS_EQUAL:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER && v1.Value.(float64) <= v2.Value.(float64) {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_GREATER_EQUAL:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER && v1.Value.(float64) >= v2.Value.(float64) {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_AND_AND:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			if v1.Type == VAR_NUMBER && v1.Value.(float64) == 0 {
				break
			}

			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v2.Type == VAR_NUMBER && v2.Value.(float64) == 0 {
				break
			}

			return MakeVariable(VAR_NUMBER, float64(1))
		}
	case TOKEN_OR_OR:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			if v1.Type != VAR_NUMBER || v1.Value.(float64) != 0 {
				return MakeVariable(VAR_NUMBER, float64(1))
			}

			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v2.Type != VAR_NUMBER || v2.Value.(float64) != 0 {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	case TOKEN_AND:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, float64(int(v1.Value.(float64))&int(v2.Value.(float64))))
			}
		}
	case TOKEN_XOR:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, float64(int(v1.Value.(float64))^int(v2.Value.(float64))))
			}
		}
	case TOKEN_OR:
		{
			v1 := ReduceVariable(Eval(tree.C[0], thread, stack))
			v2 := ReduceVariable(Eval(tree.C[1], thread, stack))
			if v1.Type == VAR_NUMBER && v2.Type == VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, float64(int(v1.Value.(float64))|int(v2.Value.(float64))))
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}
