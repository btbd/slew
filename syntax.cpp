#include "main.h"

ARRAY *GetTokens(char *s) {
	static unsigned int token_count = sizeof(SYNTAX_TOKENS) / sizeof(SYNTAX_TOKENS[0]);

	for (unsigned int i = 0; i < token_count; ++i) {
		if (SYNTAX_TOKENS[i].type != SYNTAX_TOKEN_REGEX) {
			SYNTAX_TOKENS[i].length = strlen(SYNTAX_TOKENS[i].expression);
		}
	}

	unsigned int row = 1, op = 0, cp = 0, ob = 0, cb = 0, obc = 0, cbc = 0, for_ = 0;
	char *start = s;

	int length;
	ARRAY *tokens = Array_New(sizeof(TOKEN));
	SYNTAX_TOKEN *syntax_token;
	int previous_class = TOKEN_CLASS_INIT;

	while (*s) {
		if (*s == '\n') {
			++row;
			start = s + 1;
		}

		if (*s < 33 || IsBlank(*s)) {
			++s;
			continue;
		}

		switch (*s) {
			case '(':
				++op;
				break;
			case ')':
				++cp;
				if (cp > op) {
					char *end = strchr(start, '\n');
					if (end) {
						*end = 0;
					}

					unsigned int col = s - start;
					printf("error %d:%d: unexpected token\n\t%s\n\t%*s^\n", row, col + 1, start, col, "");

					goto error;
				}

				if (for_) {
					for_ = 0;
				}
				break;
			case '[':
				++ob;
				break;
			case ']':
				++cb;
				if (cb > ob) {
					char *end = strchr(start, '\n');
					if (end) {
						*end = 0;
					}

					unsigned int col = s - start;
					printf("error %d:%d: unexpected token\n\t%s\n\t%*s^\n", row, col + 1, start, col, "");

					goto error;
				}
				break;
			case '{':
				++obc;
				break;
			case '}':
				++cbc;
				if (cbc > obc) {
					char *end = strchr(start, '\n');
					if (end) {
						*end = 0;
					}

					unsigned int col = s - start;
					printf("error %d:%d: unexpected token\n\t%s\n\t%*s^\n", row, col + 1, start, col, "");

					goto error;
				}
				break;
		}

		length = 0;
		for (unsigned int i = 0; i < token_count; ++i) {
			syntax_token = &SYNTAX_TOKENS[i];
			if (SYNTAX_TOKENS[i].type == SYNTAX_TOKEN_OPERATOR && strncmp(s, SYNTAX_TOKENS[i].expression, SYNTAX_TOKENS[i].length) == 0) {
				length = SYNTAX_TOKENS[i].length;
				break;
			} else if (SYNTAX_TOKENS[i].type == SYNTAX_TOKEN_KEYWORD && strncmp(s, SYNTAX_TOKENS[i].expression, SYNTAX_TOKENS[i].length) == 0) {
				char *n = s + SYNTAX_TOKENS[i].length;
				if (IsBlank(*n)) {
					length = SYNTAX_TOKENS[i].length;
					break;
				}

				for (unsigned int e = 0; e < token_count; ++e) {
					if (SYNTAX_TOKENS[e].type == SYNTAX_TOKEN_OPERATOR && strncmp(n, SYNTAX_TOKENS[e].expression, SYNTAX_TOKENS[e].length) == 0) {
						n = 0;
						break;
					}
				}

				if (!n) {
					length = SYNTAX_TOKENS[i].length;
					break;
				}
			} else if (SYNTAX_TOKENS[i].type == SYNTAX_TOKEN_REGEX && (length = match(s, SYNTAX_TOKENS[i].expression))) {
				break;
			}
		}

		if (!length) {
			char *end = strchr(start, '\n');
			if (end) {
				*end = 0;
			}

			unsigned int col = s - start;
			printf("error %d:%d: unexpected character\n\t%s\n\t%*s^\n", row, col + 1, start, col, "");

			goto error;
		}

		if (syntax_token->token_class != TOKEN_CLASS_IGNORE && !(!for_ && previous_class == TOKEN_CLASS_SEMICOLON && syntax_token->token_class == TOKEN_CLASS_SEMICOLON) && !(previous_class == TOKEN_CLASS_CBRACE && syntax_token->token_class == TOKEN_CLASS_SEMICOLON)) {
			if ((previous_class = SYNTAX_TABLE[previous_class][syntax_token->token_class]) == TOKEN_CLASS_ERROR) {
				char *end = strchr(start, '\n');
				if (end) {
					*end = 0;
				}

				unsigned int col = s - start;
				printf("error %d:%d: unexpected token\n\t%s\n\t%*s^\n", row, col + 1, start, col, "");

				goto error;
			}

			TOKEN token;
			token.value = (char *)calloc(length + 1, 1);
			memcpy(token.value, s, length);

			token.value_length = length;
			token.type = syntax_token->token_type;
			token.class_ = syntax_token->token_class;
			token.row = row;
			token.col = s - start + 1;

			if (token.type == TOKEN_FOR) {
				for_ = 1;
			}

			Array_Push(tokens, &token);
		}

		s += length;
	}

	if (op != cp || ob != cb || obc != cbc || SYNTAX_TABLE[previous_class][TOKEN_CLASS_END] == TOKEN_CLASS_ERROR) {
		puts("error: unexepcted end of input");
		goto error;
	}

	return tokens;

error:
	Array_Free(tokens);
	return 0;
}

int match(char *string_, char *regex_) {
	cmatch m;
	regex r(regex_);

	if (regex_search((const char *)string_, (const char *)strchr(string_, 0), m, r) && m.position() == 0) {
		return m.length();
	}

	return 0;
}