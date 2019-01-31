package main

import (
	"fmt"
	"os"
)

type Tree struct {
	T Token
	C []Tree
}

var parser_token_index int
var parser_tokens []Token
var current_token Token
var last_token Token

func ParseError(format string, args ...interface{}) {
	if current_token.Line == 0 && current_token.Col == 0 {
		fmt.Fprintf(os.Stderr, "unexpected EOF\n")
	} else {
		fmt.Fprintf(os.Stderr, "%d:%d: ", current_token.Line, current_token.Col)
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	os.Exit(1)
}

func Next() {
	parser_token_index++

	last_token = current_token

	if parser_token_index < len(parser_tokens) {
		current_token = parser_tokens[parser_token_index]
	} else {
		current_token = Token{Type: -1, Value: nil}
	}
}

func Is(token_type int) bool {
	return current_token.Type == token_type
}

func Accept(token_type int) bool {
	if Is(token_type) {
		Next()
		return true
	}

	return false
}

func Expect(token_type int) {
	if !Accept(token_type) {
		expected := ""
		for _, e := range TOKEN_PATTERNS {
			if e.Type == token_type {
				expected = e.Value.(string)
				break
			}
		}

		ParseError("Unexpected token %v expected %v", current_token.Value, expected)
	}
}

func Factor() Tree {
	if Accept(TOKEN_WORD) || Accept(TOKEN_NUMBER) || Accept(TOKEN_STRING) {
		return MakeTree(last_token)
	} else if Accept(TOKEN_OPEN_PAREN) {
		tree := Expression()
		Expect(TOKEN_CLOSE_PAREN)
		return tree
	} else if Accept(TOKEN_OPEN_BRACKET) {
		tree := MakeTree(last_token)

		e := Expression()
		for !IsNoTree(e) {
			tree.C = append(tree.C, e)
			if !Accept(TOKEN_COMMA) {
				break
			}

			e = Expression()
		}

		Expect(TOKEN_CLOSE_BRACKET)

		return tree
	} else if Accept(TOKEN_OPEN_BRACE) {
		tree := MakeTree(last_token)

		for Accept(TOKEN_WORD) {
			prop := MakeTree(last_token)
			Expect(TOKEN_COLON)
			prop.C = append(prop.C, Expression())
			tree.C = append(tree.C, prop)
			if !Accept(TOKEN_COMMA) {
				break
			}
		}

		Expect(TOKEN_CLOSE_BRACE)
		return tree
	} else if Accept(TOKEN_PLUS_PLUS) {
		last_token.Type = TOKEN_PRE_INC
		tree := MakeTree(last_token)
		tree.C = append(tree.C, Expression())
		return tree
	} else if Accept(TOKEN_MINUS_MINUS) {
		last_token.Type = TOKEN_PRE_DEC
		tree := MakeTree(last_token)
		tree.C = append(tree.C, Expression())
		return tree
	} else if Accept(TOKEN_PLUS) || Accept(TOKEN_MINUS) {
		last_token.Type = TOKEN_SIGN_PLUS + (last_token.Type-TOKEN_PLUS)/2
		tree := MakeTree(last_token)

		t := Expression()
		if IsNoTree(t) {
			ParseError(`Expected an expression after sign`)
		}

		tree.C = append(tree.C, t)
		return tree
	} else if Accept(TOKEN_NOT) || Accept(TOKEN_BIT_NOT) {
		tree := MakeTree(last_token)
		tree.C = append(tree.C, Expression())
		return tree
	} else if Accept(TOKEN_FUNC) {
		var tree Tree
		if Accept(TOKEN_WORD) {
			last_token.Type = TOKEN_FUNC
			tree = MakeTree(last_token)
		} else {
			last_token.Value = nil
			tree = MakeTree(last_token)
		}

		Expect(TOKEN_OPEN_PAREN)

		params := Tree{}

		if Accept(TOKEN_WORD) {
			params.C = append(params.C, MakeTree(last_token))

			for Accept(TOKEN_COMMA) {
				Expect(TOKEN_WORD)
				params.C = append(params.C, MakeTree(last_token))
			}
		}

		tree.C = append(tree.C, params)

		Expect(TOKEN_CLOSE_PAREN)
		Expect(TOKEN_OPEN_BRACE)
		tree.C = append(tree.C, Block())
		Expect(TOKEN_CLOSE_BRACE)

		return tree
	}

	return NoTree()
}

func Product() Tree {
	tree := Factor()

	if !IsNoTree(tree) {
		cont := true
		for cont {
			cont = false

			for Accept(TOKEN_OPEN_PAREN) {
				cont = true

				last_token.Type = TOKEN_CALL
				tree = MakeTree(last_token, tree)

				e := Expression()
				if !IsNoTree(e) {
					tree.C = append(tree.C, e)

					for Accept(TOKEN_COMMA) {
						e = Expression()
						if IsNoTree(e) {
							ParseError(`Expected an expression after comma`)
						}

						tree.C = append(tree.C, e)
					}
				}

				Expect(TOKEN_CLOSE_PAREN)
			}

			for Accept(TOKEN_OPEN_BRACKET) {
				cont = true

				last_token.Type = TOKEN_INDEX
				tree = MakeTree(last_token, tree)

				e := Expression()
				if IsNoTree(e) {
					ParseError(`Expected an expression in indexing`)
				}

				tree.C = append(tree.C, e)

				Expect(TOKEN_CLOSE_BRACKET)
			}

			for Accept(TOKEN_PERIOD) {
				cont = true
				tree = MakeTree(last_token, tree)
				Expect(TOKEN_WORD)
				tree.C = append(tree.C, MakeTree(last_token))
			}

			for Accept(TOKEN_PLUS_PLUS) {
				cont = true
				last_token.Type = TOKEN_POST_INC
				tree = MakeTree(last_token, tree)
			}

			for Accept(TOKEN_MINUS_MINUS) {
				cont = true
				last_token.Type = TOKEN_POST_DEC
				tree = MakeTree(last_token, tree)
			}

			for Accept(TOKEN_NOT) || Accept(TOKEN_BIT_NOT) {
				cont = true
				tree = MakeTree(last_token, tree)
			}
		}

		for Accept(TOKEN_MULTIPLY) || Accept(TOKEN_DIVIDE) || Accept(TOKEN_MOD) {
			tree = MakeTree(last_token, tree)

			f := Product()
			if IsNoTree(f) {
				ParseError(`Expected an expression after operator`)
			}

			tree.C = append(tree.C, f)
		}
	}

	return tree
}

func Sum() Tree {
	tree := Product()

	if !IsNoTree(tree) {
		for Accept(TOKEN_PLUS) || Accept(TOKEN_MINUS) {
			tree = MakeTree(last_token, tree)

			t := Sum()
			if IsNoTree(t) {
				ParseError(`Expected an expression after operator`)
			}

			tree.C = append(tree.C, t)
		}
	}

	return tree
}

func Comparison() Tree {
	tree := Sum()

	if !IsNoTree(tree) {
		for Accept(TOKEN_EQUAL_EQUAL) || Accept(TOKEN_NOT_EQUAL) || Accept(TOKEN_LESS) || Accept(TOKEN_GREATER) || Accept(TOKEN_LESS_EQUAL) || Accept(TOKEN_GREATER_EQUAL) {
			tree = MakeTree(last_token, tree)

			t := Comparison()
			if IsNoTree(t) {
				ParseError(`Expected an expression after operator`)
			}

			tree.C = append(tree.C, t)
		}
	}

	return tree
}

func Bitwise() Tree {
	tree := Comparison()

	if !IsNoTree(tree) {
		for Accept(TOKEN_AND) || Accept(TOKEN_XOR) || Accept(TOKEN_OR) {
			tree = MakeTree(last_token, tree)

			t := Bitwise()
			if IsNoTree(t) {
				ParseError(`Expected an expression after operator`)
			}

			tree.C = append(tree.C, t)
		}
	}

	return tree
}

func Logical() Tree {
	tree := Bitwise()

	if !IsNoTree(tree) {
		for Accept(TOKEN_AND_AND) || Accept(TOKEN_OR_OR) {
			tree = MakeTree(last_token, tree)

			t := Logical()
			if IsNoTree(t) {
				ParseError(`Expected an expression after operator`)
			}

			tree.C = append(tree.C, t)
		}
	}

	return tree
}

func Ternary() Tree {
	tree := Logical()

	if !IsNoTree(tree) {
		for Accept(TOKEN_QUESTION) {
			tree = MakeTree(last_token, tree)

			t := Ternary()
			if IsNoTree(t) {
				ParseError(`Expected an expression after ?`)
			}

			tree.C = append(tree.C, t)

			Expect(TOKEN_COLON)

			t = Ternary()
			if IsNoTree(t) {
				ParseError(`Expected an expression after :`)
			}

			tree.C = append(tree.C, t)
		}
	}

	return tree
}

func Expression() Tree {
	tree := Ternary()

	if !IsNoTree(tree) {
		for Accept(TOKEN_DECLARATION) || Accept(TOKEN_EQUAL) || Accept(TOKEN_AND_EQUAL) || Accept(TOKEN_XOR_EQUAL) || Accept(TOKEN_OR_EQUAL) || Accept(TOKEN_SHIFT_LEFT_EQUAL) || Accept(TOKEN_SHIFT_RIGHT_EQUAL) || Accept(TOKEN_PLUS_EQUAL) || Accept(TOKEN_MINUS_EQUAL) || Accept(TOKEN_MULTIPLY_EQUAL) || Accept(TOKEN_DIVIDE_EQUAL) || Accept(TOKEN_MOD_EQUAL) {
			compound := (last_token.Type != TOKEN_DECLARATION && last_token.Type != TOKEN_EQUAL)
			if compound {
				last_token.Type++
				last_token.Value = last_token.Value.(string)[0 : len(last_token.Value.(string))-1]
				op := last_token
				ov := tree
				tree = MakeTree(Token{Type: TOKEN_EQUAL, Value: "=", Col: last_token.Col, Line: last_token.Line}, tree)

				t := Expression()
				if IsNoTree(t) {
					ParseError(`Expected an expression after operator`)
				}

				tree.C = append(tree.C, MakeTree(op, ov, t))
			} else {
				tree = MakeTree(last_token, tree)

				t := Expression()
				if IsNoTree(t) {
					ParseError(`Expected an expression after operator`)
				}

				tree.C = append(tree.C, t)
			}
		}
	}

	return tree
}

func Statement() Tree {
	for Accept(TOKEN_SEMICOLON) {
	}

	if Accept(TOKEN_IF) {
		return IfTree()
	} else if Accept(TOKEN_FOR) {
		tree := MakeTree(last_token)

		Expect(TOKEN_OPEN_PAREN)

		init := MakeTree(Token{Type: TOKEN_BLOCK}, Expression())
		for Accept(TOKEN_COMMA) {
			e := Expression()
			if IsNoTree(e) {
				ParseError(`expected expression after ,`)
			}

			init.C = append(init.C, e)
		}
		tree.C = append(tree.C, init)

		Expect(TOKEN_SEMICOLON)
		tree.C = append(tree.C, Expression())
		Expect(TOKEN_SEMICOLON)

		inc := MakeTree(Token{Type: TOKEN_BLOCK}, Expression())
		for Accept(TOKEN_COMMA) {
			e := Expression()
			if IsNoTree(e) {
				ParseError(`expected expression after ,`)
			}

			inc.C = append(inc.C, e)
		}
		tree.C = append(tree.C, inc)

		Expect(TOKEN_CLOSE_PAREN)
		Expect(TOKEN_OPEN_BRACE)
		tree.C = append(tree.C, Block())
		Expect(TOKEN_CLOSE_BRACE)

		return tree
	} else if Accept(TOKEN_WHILE) {
		tree := MakeTree(last_token)

		Expect(TOKEN_OPEN_PAREN)

		condition := Expression()
		if IsNoTree(condition) {
			ParseError(`while condition cannot be empty`)
		}

		tree.C = append(tree.C, condition)

		Expect(TOKEN_CLOSE_PAREN)
		Expect(TOKEN_OPEN_BRACE)
		tree.C = append(tree.C, Block())
		Expect(TOKEN_CLOSE_BRACE)

		return tree
	} else if Accept(TOKEN_BREAK) {
		tree := MakeTree(last_token)

		e := Expression()
		if !IsNoTree(e) {
			ParseError(`error: unexpected expression`)
		}

		return tree
	} else if Accept(TOKEN_CONTINUE) {
		return MakeTree(last_token)
	} else if Accept(TOKEN_RETURN) {
		tree := MakeTree(last_token)

		e := Expression()
		if !IsNoTree(e) {
			tree.C = append(tree.C, e)
		}

		return tree
	}

	return Expression()
}

func Block() Tree {
	tree := MakeTree(Token{Type: TOKEN_BLOCK})

	for parser_token_index < len(parser_tokens) {
		s := Statement()
		if IsNoTree(s) {
			break
		}

		tree.C = append(tree.C, s)
	}

	return tree
}

func IfTree() Tree {
	tree := MakeTree(last_token)

	condition := Expression()
	if IsNoTree(condition) {
		ParseError("expected an expression after if")
	}

	Expect(TOKEN_OPEN_BRACE)
	tree.C = append(tree.C, condition, Block())
	Expect(TOKEN_CLOSE_BRACE)

	if Accept(TOKEN_ELSE) {
		if Accept(TOKEN_IF) {
			tree.C = append(tree.C, IfTree())
		} else {
			Expect(TOKEN_OPEN_BRACE)
			tree.C = append(tree.C, Block())
			Expect(TOKEN_CLOSE_BRACE)
		}
	}

	return tree
}

func IsNoTree(tree Tree) bool {
	return tree.T.Type == -1
}

func NoTree() Tree {
	return Tree{T: Token{Type: -1}}
}

func MakeTree(token Token, children ...Tree) Tree {
	return Tree{T: token, C: children}
}

func Parse(tokens []Token) Tree {
	parser_tokens = tokens
	parser_token_index = -1
	Next()
	tree := Block()
	if parser_token_index < len(parser_tokens) {
		ParseError("unexpected token: %v", current_token.Value)
	}
	return tree
}

func PrintTree(tree Tree, depth int) {
	for i := 0; i < depth; i++ {
		fmt.Print("  ")
	}

	fmt.Println(tree.T.Value)

	depth++
	for _, e := range tree.C {
		PrintTree(e, depth)
	}
}
