package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	TOKEN_IGNORE int = iota

	TOKEN_DECLARATION

	TOKEN_COMMA
	TOKEN_QUESTION
	TOKEN_COLON
	TOKEN_PERIOD

	TOKEN_EQUAL_EQUAL
	TOKEN_NOT_EQUAL
	TOKEN_EQUAL

	TOKEN_AND_AND
	TOKEN_AND_EQUAL
	TOKEN_AND

	TOKEN_XOR_EQUAL
	TOKEN_XOR

	TOKEN_OR_OR
	TOKEN_OR_EQUAL
	TOKEN_OR

	TOKEN_BIT_NOT
	TOKEN_NOT

	TOKEN_LESS_EQUAL
	TOKEN_SHIFT_LEFT_EQUAL
	TOKEN_SHIFT_LEFT
	TOKEN_LESS

	TOKEN_GREATER_EQUAL
	TOKEN_SHIFT_RIGHT_EQUAL
	TOKEN_SHIFT_RIGHT
	TOKEN_GREATER

	TOKEN_PLUS_PLUS
	TOKEN_MINUS_MINUS

	TOKEN_PLUS_EQUAL
	TOKEN_PLUS
	TOKEN_MINUS_EQUAL
	TOKEN_MINUS

	TOKEN_MULTIPLY_EQUAL
	TOKEN_MULTIPLY

	TOKEN_DIVIDE_EQUAL
	TOKEN_DIVIDE

	TOKEN_MOD_EQUAL
	TOKEN_MOD

	TOKEN_OPEN_PAREN
	TOKEN_CLOSE_PAREN

	TOKEN_SEMICOLON

	TOKEN_OPEN_BRACKET
	TOKEN_CLOSE_BRACKET

	TOKEN_OPEN_BRACE
	TOKEN_CLOSE_BRACE

	TOKEN_FUNC
	TOKEN_IF
	TOKEN_ELSE
	TOKEN_FOR
	TOKEN_WHILE
	TOKEN_BREAK
	TOKEN_CONTINUE
	TOKEN_RETURN

	TOKEN_WORD
	TOKEN_NUMBER
	TOKEN_STRING

	TOKEN_BLOCK
	TOKEN_CALL
	TOKEN_INDEX
	TOKEN_PRE_INC
	TOKEN_POST_INC
	TOKEN_PRE_DEC
	TOKEN_POST_DEC
	TOKEN_SIGN_PLUS
	TOKEN_SIGN_MINUS
)

type Token struct {
	Type  int
	Value interface{}
	Line  int
	Col   int
}

var TOKEN_PATTERNS = []Token{
	{TOKEN_IGNORE, `\/\/.*`, 0, 0},
	{TOKEN_IGNORE, `\/\*[\S\s]*\*\/`, 0, 0},
	{TOKEN_IGNORE, `\\r\n`, 0, 0},
	{TOKEN_IGNORE, `\\n`, 0, 0},
	{TOKEN_IGNORE, `\s`, 0, 0},

	{TOKEN_DECLARATION, `:=`, 0, 0},
	{TOKEN_COMMA, `,`, 0, 0},
	{TOKEN_QUESTION, `\?`, 0, 0},
	{TOKEN_COLON, `:`, 0, 0},
	{TOKEN_PERIOD, `\.`, 0, 0},

	{TOKEN_EQUAL_EQUAL, `==`, 0, 0},
	{TOKEN_NOT_EQUAL, `!=`, 0, 0},
	{TOKEN_EQUAL, `=`, 0, 0},

	{TOKEN_AND_AND, `&&`, 0, 0},
	{TOKEN_AND_EQUAL, `&=`, 0, 0},
	{TOKEN_AND, `&`, 0, 0},

	{TOKEN_XOR_EQUAL, `\^=`, 0, 0},
	{TOKEN_XOR, `\^`, 0, 0},

	{TOKEN_OR_OR, `\|\|`, 0, 0},
	{TOKEN_OR_EQUAL, `\|=`, 0, 0},
	{TOKEN_OR, `\|`, 0, 0},

	{TOKEN_BIT_NOT, `~`, 0, 0},
	{TOKEN_NOT, `!`, 0, 0},

	{TOKEN_LESS_EQUAL, `<=`, 0, 0},
	{TOKEN_SHIFT_LEFT_EQUAL, `<<=`, 0, 0},
	{TOKEN_SHIFT_LEFT, `<<`, 0, 0},
	{TOKEN_LESS, `<`, 0, 0},

	{TOKEN_GREATER_EQUAL, `>=`, 0, 0},
	{TOKEN_SHIFT_RIGHT_EQUAL, `>>=`, 0, 0},
	{TOKEN_SHIFT_RIGHT, `>>`, 0, 0},
	{TOKEN_GREATER, `>`, 0, 0},

	{TOKEN_PLUS_PLUS, `\+\+`, 0, 0},
	{TOKEN_PLUS_EQUAL, `\+=`, 0, 0},
	{TOKEN_PLUS, `\+`, 0, 0},

	{TOKEN_MINUS_MINUS, `--`, 0, 0},
	{TOKEN_MINUS_EQUAL, `-=`, 0, 0},
	{TOKEN_MINUS, `-`, 0, 0},

	{TOKEN_MULTIPLY_EQUAL, `\*=`, 0, 0},
	{TOKEN_MULTIPLY, `\*`, 0, 0},

	{TOKEN_DIVIDE_EQUAL, `/=`, 0, 0},
	{TOKEN_DIVIDE, `/`, 0, 0},

	{TOKEN_MOD_EQUAL, `%=`, 0, 0},
	{TOKEN_MOD, `%`, 0, 0},

	{TOKEN_OPEN_PAREN, `\(`, 0, 0},
	{TOKEN_CLOSE_PAREN, `\)`, 0, 0},

	{TOKEN_SEMICOLON, `;`, 0, 0},

	{TOKEN_OPEN_BRACKET, `\[`, 0, 0},
	{TOKEN_CLOSE_BRACKET, `\]`, 0, 0},

	{TOKEN_OPEN_BRACE, `{`, 0, 0},
	{TOKEN_CLOSE_BRACE, `}`, 0, 0},

	{TOKEN_FUNC, `(func)(\W|$)`, 0, 0},
	{TOKEN_IF, `(if)(\W|$)`, 0, 0},
	{TOKEN_ELSE, `(else)(\W|$)`, 0, 0},
	{TOKEN_FOR, `(for)(\W|$)`, 0, 0},
	{TOKEN_WHILE, `(while)(\W|$)`, 0, 0},
	{TOKEN_BREAK, `(break)(\W|$)`, 0, 0},
	{TOKEN_CONTINUE, `(continue)(\W|$)`, 0, 0},
	{TOKEN_RETURN, `(return)(\W|$)`, 0, 0},

	{TOKEN_WORD, `([a-zA-Z_$][a-zA-Z0-9_$]+)|([a-zA-Z_$]+)`, 0, 0},

	{TOKEN_NUMBER, `0[0-7]+`, 0, 0},
	{TOKEN_NUMBER, `0b[0|1]+`, 0, 0},
	{TOKEN_NUMBER, `0x[a-fA-F0-9]+`, 0, 0},
	{TOKEN_NUMBER, `(([0-9]+\.[0-9]+)|([0-9]+)|(\.[0-9]+))e([0-9]+)`, 0, 0},
	{TOKEN_NUMBER, `(([0-9]+\.[0-9]+)|([0-9]+)|(\.[0-9]+))E([0-9]+)`, 0, 0},
	{TOKEN_NUMBER, `([0-9]+\.[0-9]+)|([0-9]+)|(\.[0-9]+)`, 0, 0},

	{TOKEN_STRING, `"(?:[^"\\]|\\.)*"`, 0, 0},
	{TOKEN_STRING, `'(?:[^'\\]|\\.)*'`, 0, 0},
}

var STATIC_ESCAPES = []struct {
	m string
	r string
}{
	{`\\`, "\\"},
	{`\a`, "\a"},
	{`\b`, "\b"},
	{`\f`, "\f"},
	{`\n`, "\n"},
	{`\r`, "\r"},
	{`\t`, "\t"},
	{`\v`, "\v"},
}

func Tokenize(input string) []Token {
	var tokens []Token

	line := 1
	col := 1

	for len(input) > 0 {
		current := input
		for _, e := range TOKEN_PATTERNS {
			re := regexp.MustCompile("^(" + e.Value.(string) + ")")

			match := ""

			// Go doesn't support regex lookaheads
			if e.Type >= TOKEN_FUNC && e.Type <= TOKEN_RETURN {
				m := re.FindStringSubmatch(input)
				if len(m) > 2 {
					match = m[2]
				}
			} else {
				match = re.FindString(input)
			}

			if match != "" {
				if e.Type != TOKEN_IGNORE {
					var value interface{}

					if e.Type == TOKEN_NUMBER {
						if strings.IndexByte(match, '.') != -1 {
							v, err := strconv.ParseFloat(match, 64)
							if err != nil {
								fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, err.Error())
								return []Token{}
							}

							value = v
						} else if exp := strings.IndexByte(strings.ToLower(match), 'e'); exp != -1 && strings.IndexByte(strings.ToLower(match), 'x') == -1 {
							v, err := strconv.ParseFloat(match[:exp], 64)
							if err != nil {
								fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, err.Error())
								return []Token{}
							}

							p, err := strconv.ParseInt(match[exp+1:], 0, 64)
							if err != nil {
								fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, err.Error())
								return []Token{}
							}

							value = v * math.Pow(10, float64(p))
						} else {
							base := 0
							trim := match
							if strings.HasPrefix(match, "0b") {
								trim = trim[2:]
								base = 2
							} else if len(trim) > 1 && trim[0] == '0' && trim[1] != 'x' {
								trim = trim[1:]
								base = 8
							}

							v, err := strconv.ParseInt(trim, base, 64)
							if err != nil {
								fmt.Fprintf(os.Stderr, "%d:%d: %s\n", line, col, err.Error())
								return []Token{}
							}

							value = float64(v)
						}
					} else if e.Type == TOKEN_STRING {
						q := `\` + string(match[0])
						s := []rune(match[1 : len(match)-1])
						r := ""

					loop:
						for i := 0; i < len(s); i++ {
							c := string(s[i:])
							if strings.HasPrefix(c, q) {
								r += string(q[1])
								i++
								continue
							}

							if strings.HasPrefix(c, `\x`) {
								i += 2
								if i+1 >= len(s) {
									fmt.Fprintf(os.Stderr, "%d:%d: Unexpected end of string: %s\n", line, col, match)
									return []Token{}
								}

								c = string(s[i:])

								var b byte
								fmt.Sscanf(c, "%02x", &b)
								r += string(rune(b))
								i++
								continue
							}

							if strings.HasPrefix(c, `\u`) {
								i += 2
								if i+3 >= len(s) {
									fmt.Fprintf(os.Stderr, "%d:%d: Unexpected end of string: %s\n", line, col, match)
									return []Token{}
								}

								c = string(s[i:])

								b := 0
								fmt.Sscanf(c, "%04x", &b)
								r += string(rune(b))
								i += 3
								continue
							}

							for _, e := range STATIC_ESCAPES {
								if strings.HasPrefix(c, e.m) {
									r += e.r
									i++
									continue loop
								}
							}

							r += string(s[i])
						}
						value = r
					} else {
						value = match
					}

					tokens = append(tokens, Token{Type: e.Type, Value: value, Line: line, Col: col})
				}

				if n := strings.Count(match, "\n"); n > 0 {
					line += n
					col = 0
				}

				col += len(match)

				input = input[len(match):]
				break
			}
		}

		if input == current {
			fmt.Fprintf(os.Stderr, "%d:%d: Unexpected token: %s\n", line, col, strings.Split(input, "\n")[0])
			return []Token{}
		}
	}

	return tokens
}
