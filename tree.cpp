#include "main.h"

TOKEN *CopyToken(TOKEN *token) {
	TOKEN *t = (TOKEN *)calloc(sizeof(TOKEN), 1);
	memcpy(t, token, sizeof(TOKEN));
	t->value = _strdup(token->value);
	return t;
}

void FreeToken(TOKEN *token) {
	if (token->value) {
		free(token->value);
	}
	free(token);
}

TREE *CreateTree(ARRAY *tokens, unsigned int *i, bool sign) {
	ARRAY expression_stack = ArrayNew(sizeof(TREE));
	ARRAY operator_stack = ArrayNew(sizeof(TOKEN *));

	TOKEN *token = 0;
	TOKEN *token_previous = 0;

	for (; *i < tokens->length; ++(*i)) {
		if (sign && operator_stack.length > 0) {
			--(*i);
			goto leave;
		}
		token = (TOKEN *)ArrayGet(tokens, *i);

		switch (token->class_) {
			case TOKEN_CLASS_IGNORE:
				continue;
			case TOKEN_CLASS_OPAREN: {
				if (token_previous && token_previous->class_ == TOKEN_CLASS_WORD) {
					TREE call = { 0 };

					call.left = (TREE *)calloc(sizeof(TREE), 1);
					call.token = (TOKEN *)calloc(sizeof(TREE), 1);
					call.token->value = _strdup("()");
					call.token->value_length = 2;
					call.token->type = TOKEN_CALL;
					token = call.token;
					ArrayPop(&expression_stack, call.left);

					TOKEN *comma = (TOKEN *)calloc(sizeof(TOKEN), 1);
					comma->value = _strdup(",");
					comma->value_length = 1;
					comma->class_ = TOKEN_CLASS_COMMA;
					comma->type = TOKEN_COMMA;

					TREE *tree = &call;
					TOKEN *t;
					for (++(*i);; ++(*i)) {
						tree = (tree->right = (TREE *)calloc(sizeof(TREE), 1));
						tree->token = CopyToken(comma);
						tree->left = CreateTree(tokens, i, false);

						if (!tree->left) {
							break;
						}

						t = (TOKEN *)ArrayGet(tokens, *i);
						if (t->type == TOKEN_CLOSE_PAREN) {
							break;
						} else if (t->type != TOKEN_COMMA) {
							printf("error %d:%d: unexpected token (expected ',' or ')')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}
					}

					FreeToken(comma);
					ArrayPush(&expression_stack, &call);
				} else {
					++(*i);
					TREE *e = CreateTree(tokens, i, false);
					if (!e) {
						goto error;
					}

					ArrayPush(&expression_stack, e);
				}

				break;
			}
			case TOKEN_CLASS_CPAREN: case TOKEN_CLASS_COMMA: case TOKEN_CLASS_SEMICOLON: case TOKEN_CLASS_CBRACE: case TOKEN_CLASS_CBRACKET: case TOKEN_CLASS_COLON:
				goto leave;
			case TOKEN_CLASS_QUESTION: {
				if (expression_stack.length > 0) {
					TREE e;
					e.token = CopyToken(token);
					e.left = (TREE *)calloc(sizeof(TREE), 1);

					while (operator_stack.length > 0) {
						TOKEN *o = (*(TOKEN **)ArrayGet(&operator_stack, operator_stack.length - 1));
						if (o->class_ == TOKEN_CLASS_BINARY && expression_stack.length > 1) {
							TREE *e1 = (TREE *)malloc(sizeof(TREE));
							TREE *e2 = (TREE *)malloc(sizeof(TREE));

							ArrayPop(&expression_stack, e2);
							ArrayPop(&expression_stack, e1);

							TREE expression;
							ArrayPop(&operator_stack, &expression.token);
							expression.token = CopyToken(expression.token);
							expression.left = e1;
							expression.right = e2;

							ArrayPush(&expression_stack, &expression);
						} else {
							TREE *e = (TREE *)malloc(sizeof(TREE));

							ArrayPop(&expression_stack, e);

							TREE expression;
							ArrayPop(&operator_stack, &expression.token);
							expression.token = CopyToken(expression.token);
							expression.left = e;
							expression.right = 0;

							ArrayPush(&expression_stack, &expression);
						}
					}

					ArrayPop(&expression_stack, e.left);
					e.right = (TREE *)calloc(sizeof(TREE), 1);

					e.right->token = (TOKEN *)calloc(sizeof(TOKEN), 1);
					e.right->token->value = _strdup(",");
					e.right->token->value_length = 1;
					e.right->token->class_ = TOKEN_CLASS_COMMA;
					e.right->token->type = TOKEN_COMMA;

					++(*i);
					e.right->left = CreateTree(tokens, i, false);
					TOKEN *t = (TOKEN *)ArrayGet(tokens, *i);
					if (!e.right->left || t->class_ != TOKEN_CLASS_COLON) {
						printf("error %d:%d: unexpected token (expected ':')\n\t%s\n\t^\n", t->row, t->col, t->value);
						goto error;
					}

					++(*i);
					e.right->right = CreateTree(tokens, i, false);
					--(*i);

					ArrayPush(&expression_stack, &e);
				} else {
					printf("error %d:%d: unexpected token\n\t%s\n\t^\n", token->row, token->col, token->value);
					goto error;
				}

				break;
			}
			case TOKEN_CLASS_UNARY: {
				TREE e;
				e.token = CopyToken(token);
				++(*i);
				e.left = CreateTree(tokens, i, false);
				--(*i);
				if (!e.left) goto error;
				e.right = 0;
				ArrayPush(&expression_stack, &e);

				break;
			}
			case TOKEN_CLASS_SIGN: {
				if (token_previous && (token_previous->class_ == TOKEN_CLASS_WORD ||
					token_previous->class_ == TOKEN_CLASS_KEYWORD ||
					token_previous->class_ == TOKEN_CLASS_NUMBER ||
					token_previous->class_ == TOKEN_CLASS_STRING ||
					token_previous->type == TOKEN_CALL ||
					token_previous->class_ == TOKEN_CLASS_CPAREN ||
					token_previous->class_ == TOKEN_CLASS_CBRACKET
					)) {
					goto binary;
				}

				TREE e;
				e.token = CopyToken(token);
				e.left = (TREE *)malloc(sizeof(TREE));
				e.left->left = e.left->right = 0;

				e.left->token = (TOKEN *)malloc(sizeof(TOKEN));
				e.left->token->class_ = TOKEN_CLASS_NUMBER;
				e.left->token->type = TOKEN_DECIMAL_NUMBER;
				e.left->token->value = _strdup("0");
				e.left->token->value_length = 1;

				++(*i);
				e.right = CreateTree(tokens, i, true);
				if (!e.right) goto error;
				--(*i);

				ArrayPush(&expression_stack, &e);

				break;
			}
			case TOKEN_CLASS_INC_DEC:
				if (sign) goto leave;
				if (token_previous && token_previous->class_ == TOKEN_CLASS_WORD) {
					*token->value = '>';
				}
			case TOKEN_CLASS_BINARY: {
			binary:
				token->class_ = TOKEN_CLASS_BINARY;
				while (operator_stack.length > 0 && TOKEN_PRECEDENCE[(*(TOKEN **)ArrayGet(&operator_stack, operator_stack.length - 1))->type] >= TOKEN_PRECEDENCE[token->type]) {
					TOKEN *o = (*(TOKEN **)ArrayGet(&operator_stack, operator_stack.length - 1));
					if (o->class_ == TOKEN_CLASS_BINARY && expression_stack.length > 1) {
						TREE *e1 = (TREE *)malloc(sizeof(TREE));
						TREE *e2 = (TREE *)malloc(sizeof(TREE));

						ArrayPop(&expression_stack, e2);
						ArrayPop(&expression_stack, e1);

						TREE expression;
						ArrayPop(&operator_stack, &expression.token);
						expression.token = CopyToken(expression.token);
						expression.left = e1;
						expression.right = e2;

						ArrayPush(&expression_stack, &expression);
					} else {
						TREE *e = (TREE *)malloc(sizeof(TREE));

						ArrayPop(&expression_stack, e);

						TREE expression;
						ArrayPop(&operator_stack, &expression.token);
						expression.token = CopyToken(expression.token);
						expression.left = e;
						expression.right = 0;

						ArrayPush(&expression_stack, &expression);
					}
				}

				ArrayPush(&operator_stack, &token);

				break;
			}
			case TOKEN_CLASS_WORD: {
				TREE expression;
				expression.left = expression.right = 0;

				if (strchr(token->value, '.')) {
					TREE *e = &expression;
					char *s = token->value;
					char *n = 0;
					while ((n = strchr(s, '.'))) {
						e->token = (TOKEN *)calloc(sizeof(TOKEN), 1);
						e->token->type = TOKEN_WORD;
						e->token->value_length = (unsigned int)((SINT)n - (SINT)s);
						e->token->value = (char *)calloc(e->token->value_length + 1, 1);
						memcpy(e->token->value, s, e->token->value_length);
						e = (e->left = (TREE *)calloc(sizeof(TREE), 1));

						s = n + 1;
					}

					e->token = (TOKEN *)calloc(sizeof(TOKEN), 1);
					e->token->type = TOKEN_WORD;
					e->token->value_length = (unsigned int)strlen(s);
					e->token->value = _strdup(s);
				} else {
					expression.token = CopyToken(token);
				}

				ArrayPush(&expression_stack, &expression);

				break;
			}
			case TOKEN_CLASS_NUMBER: case TOKEN_CLASS_STRING: {
				TREE expression;
				expression.token = CopyToken(token);
				expression.left = expression.right = 0;

				ArrayPush(&expression_stack, &expression);

				break;
			}
			case TOKEN_CLASS_OBRACKET: {
				if (((TOKEN *)ArrayGet(tokens, *i + 1))->type == TOKEN_CLOSE_BRACKET) {
					TREE expression;
					token->type = TOKEN_ARRAY;
					expression.token = CopyToken(token);
					expression.left = expression.right = 0;

					ArrayPush(&expression_stack, &expression);

					++(*i);
				} else {
					TREE e;
					token->type = TOKEN_ARRAY;
					e.token = CopyToken(token);
					++(*i);
					e.left = CreateTree(tokens, i, false);
					--(*i);
					if (!e.left) goto error;
					e.right = 0;

					TREE *arg = &e;
					unsigned int c = 1;
					while (c > 0) {
						token = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (token->type == TOKEN_OPEN_BRACKET) {
							++c;
						} else if (token->type == TOKEN_CLOSE_BRACKET) {
							--c;
						} else if (token->type == TOKEN_COMMA) {
							arg = (arg->right = (TREE *)calloc(sizeof(TREE), 1));

							arg->token = CopyToken(token);
							++(*i);
							arg->left = CreateTree(tokens, i, false);
							--(*i);
							if (!arg->left) goto error;
							arg->right = 0;
						}
					}

					ArrayPush(&expression_stack, &e);
				}

				break;
			}
			case TOKEN_CLASS_KEYWORD: {
				switch (token->type) {
					case TOKEN_FUNC: {
						if (*i + 3 >= tokens->length) {
							puts("error: unexepcted end of input");
							goto error;
						}

						TREE func;
						func.token = CopyToken(token);
						func.left = func.right = 0;

						TOKEN *t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_PAREN) {
							printf("error %d:%d: unexpected token (expected '(')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						TREE *arg = &func;
						while (t->type != TOKEN_CLOSE_PAREN) {
							t = (TOKEN *)ArrayGet(tokens, ++(*i));
							if (t->type == TOKEN_CLOSE_PAREN) {
								break;
							} else if (t->class_ != TOKEN_CLASS_WORD) {
								printf("error %d:%d: unexpected token (expected ')' or an argument)\n\t%s\n\t^\n", t->row, t->col, t->value);
								goto error;
							}

							arg = (arg->left = (TREE *)malloc(sizeof(TREE)));
							arg->token = CopyToken(t);
							arg->left = arg->right = 0;

							t = (TOKEN *)ArrayGet(tokens, ++(*i));
							if (t->type != TOKEN_COMMA && t->type != TOKEN_CLOSE_PAREN) {
								printf("error %d:%d: unexpected token (expected ',' or ')')\n\t%s\n\t^\n", t->row, t->col, t->value);
								goto error;
							}
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_BRACE) {
							printf("error %d:%d: unexpected token (expected '{')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_CLOSE_BRACE) {
							TOKEN *comma = (TOKEN *)calloc(sizeof(TOKEN), 1);
							comma->value = _strdup(",");
							comma->value_length = 1;
							comma->class_ = TOKEN_CLASS_COMMA;
							comma->type = TOKEN_COMMA;

							TREE *statement = &func;
							for (;;) {
								statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
								statement->token = CopyToken(comma);
								statement->left = CreateTree(tokens, i, false);

								t = (TOKEN *)ArrayGet(tokens, *i);
								if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
									if (!statement->left) {
										break;
									}

									t = (TOKEN *)ArrayGet(tokens, ++(*i));
									if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
										break;
									}
									continue;
								}

								if (!statement->left) {
									goto error;
								}

								break;
							}

							FreeToken(comma);
						}

						t = (TOKEN *)ArrayGet(tokens, *i);
						ArrayPush(&expression_stack, &func);

						goto leave;
					}
					case TOKEN_IF: {
						if (*i + 4 >= tokens->length) {
							puts("error: unexepcted end of input");
							goto error;
						}

						TOKEN *if_ = token;

						TOKEN *t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_PAREN) {
							printf("error %d:%d: unexpected token (expected '(')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type == TOKEN_CLOSE_PAREN) {
							printf("error %d:%d: expected an expression\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						TOKEN *comma = (TOKEN *)calloc(sizeof(TOKEN), 1);
						comma->value = _strdup(",");
						comma->value_length = 1;
						comma->class_ = TOKEN_CLASS_COMMA;
						comma->type = TOKEN_COMMA;

						TREE expression;
						expression.token = CopyToken(token);
						expression.left = (TREE *)calloc(sizeof(TREE), 1);
						expression.left->token = CopyToken(comma);
						expression.left->left = CreateTree(tokens, i, false);
						if (!expression.left->left) goto error;
						expression.right = 0;

						t = (TOKEN *)ArrayGet(tokens, *i);
						if (t->type != TOKEN_CLOSE_PAREN) {
							printf("error %d:%d: unexpected token (expected ')')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_BRACE) {
							printf("error %d:%d: unexpected token (expected '{')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_CLOSE_BRACE) {
							TREE *statement = expression.left;
							for (;;) {
								statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
								statement->token = CopyToken(comma);
								statement->left = CreateTree(tokens, i, false);

								t = (TOKEN *)ArrayGet(tokens, *i);
								if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
									if (!statement->left) {
										break;
									}

									t = (TOKEN *)ArrayGet(tokens, ++(*i));
									if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
										break;
									}
									continue;
								}

								if (!statement->left) {
									goto error;
								}

								break;
							}
						}

						if (*i + 1 < tokens->length) {
							t = (TOKEN *)ArrayGet(tokens, *i + 1);
							if (t->type == TOKEN_ELSE_IF || t->type == TOKEN_ELSE) {
								++(*i);
								TREE *block = &expression;

								for (;; t = (TOKEN *)ArrayGet(tokens, ++(*i))) {
									if (t->type == TOKEN_ELSE_IF) {
										if (*i + 4 >= tokens->length) {
											puts("error: unexepcted end of input");
											goto error;
										}

										TOKEN *elseif = t;

										t = (TOKEN *)ArrayGet(tokens, ++(*i));
										if (t->type != TOKEN_OPEN_PAREN) {
											printf("error %d:%d: unexpected token (expected '(')\n\t%s\n\t^\n", t->row, t->col, t->value);
											goto error;
										}

										t = (TOKEN *)ArrayGet(tokens, ++(*i));
										if (t->type == TOKEN_CLOSE_PAREN) {
											printf("error %d:%d: expected an expression\n\t%s\n\t^\n", t->row, t->col, t->value);
											goto error;
										}

										TREE *expression = (TREE *)calloc(sizeof(TREE), 1);
										expression->token = CopyToken(elseif);
										expression->left = (TREE *)calloc(sizeof(TREE), 1);
										expression->left->token = CopyToken(comma);
										expression->left->left = CreateTree(tokens, i, false);
										if (!expression->left->left) goto error;

										t = (TOKEN *)ArrayGet(tokens, *i);
										if (t->type != TOKEN_CLOSE_PAREN) {
											printf("error %d:%d: unexpected token (expected ')')\n\t%s\n\t^\n", t->row, t->col, t->value);
											goto error;
										}

										t = (TOKEN *)ArrayGet(tokens, ++(*i));
										if (t->type != TOKEN_OPEN_BRACE) {
											printf("error %d:%d: unexpected token (expected '{')\n\t%s\n\t^\n", t->row, t->col, t->value);
											goto error;
										}

										t = (TOKEN *)ArrayGet(tokens, ++(*i));
										if (t->type != TOKEN_CLOSE_BRACE) {
											TREE *statement = expression->left;
											for (;;) {
												statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
												statement->token = CopyToken(comma);
												statement->left = CreateTree(tokens, i, false);

												t = (TOKEN *)ArrayGet(tokens, *i);
												if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
													if (!statement->left) {
														break;
													}

													t = (TOKEN *)ArrayGet(tokens, ++(*i));
													if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
														--(*i);
														break;
													}
													continue;
												}

												if (!statement->left) {
													goto error;
												}

												break;
											}
										}

										block = (block->right = expression);
									} else if (t->type == TOKEN_ELSE) {
										if (*i + 2 >= tokens->length) {
											puts("error: unexepcted end of input");
											goto error;
										}

										TOKEN *else_ = t;

										t = (TOKEN *)ArrayGet(tokens, ++(*i));
										if (t->type != TOKEN_OPEN_BRACE) {
											printf("error %d:%d: unexpected token (expected '{')\n\t%s\n\t^\n", t->row, t->col, t->value);
											goto error;
										}

										TREE *expression = (TREE *)calloc(sizeof(TREE), 1);
										expression->token = CopyToken(else_);

										t = (TOKEN *)ArrayGet(tokens, ++(*i));
										if (t->type != TOKEN_CLOSE_BRACE) {
											TREE *statement = expression;
											for (;;) {
												statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
												statement->token = CopyToken(comma);
												statement->left = CreateTree(tokens, i, false);

												t = (TOKEN *)ArrayGet(tokens, *i);
												if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
													if (!statement->left) {
														break;
													}

													t = (TOKEN *)ArrayGet(tokens, ++(*i));
													if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
														--(*i);
														break;
													}
													continue;
												}

												if (!statement->left) {
													goto error;
												}

												break;
											}
										}

										block = (block->right = expression);
										break;
									} else {
										break;
									}
								}
							}
						}

						FreeToken(comma);
						ArrayPush(&expression_stack, &expression);

						goto leave;
					}
					case TOKEN_ELSE_IF:
						printf("error %d:%d: unexpected token 'else if'\n\t%s\n\t^\n", token->row, token->col, token->value);
						goto error;
					case TOKEN_ELSE:
						printf("error %d:%d: unexpected token 'else'\n\t%s\n\t^\n", token->row, token->col, token->value);
						goto error;
					case TOKEN_FOR: {
						if (*i + 4 >= tokens->length) {
							puts("error: unexepcted end of input");
							goto error;
						}

						TOKEN *for_ = (TOKEN *)ArrayGet(tokens, *i);
						TOKEN *t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_PAREN) {
							printf("error %d:%d: unexpected token (expected '(')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type == TOKEN_CLOSE_PAREN) {
							printf("error %d:%d: expected an expression\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						TOKEN *comma = (TOKEN *)calloc(sizeof(TOKEN), 1);
						comma->value = _strdup(",");
						comma->value_length = 1;
						comma->class_ = TOKEN_CLASS_COMMA;
						comma->type = TOKEN_COMMA;

						TREE expression;
						expression.token = CopyToken(for_);
						expression.left = (TREE *)calloc(sizeof(TREE), 1);
						expression.right = 0;

						TREE *statement = expression.left;
						while ((t = (TOKEN *)ArrayGet(tokens, *i))->type != TOKEN_SEMICOLON) {
							statement->token = CopyToken(comma);
							statement->left = CreateTree(tokens, i, false);

							t = (TOKEN *)ArrayGet(tokens, *i);
							if (t->type == TOKEN_COMMA) {
								statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
								++(*i);
								continue;
							} else if (t->type != TOKEN_SEMICOLON) {
								printf("error %d:%d: unexpected token (expected ',' or ';')\n\t%s\n\t^\n", t->row, t->col, t->value);
								goto error;
							}
						}

						if (*i + 3 >= tokens->length) {
							puts("error: unexepcted end of input");
							goto error;
						}

						statement = (expression.right = (TREE *)calloc(sizeof(TREE), 1));
						statement->token = CopyToken(comma);

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_SEMICOLON) {
							statement->left = CreateTree(tokens, i, false);

							t = (TOKEN *)ArrayGet(tokens, *i);
							if (t->type != TOKEN_SEMICOLON) {
								printf("error %d:%d: unexpected token (expected ';')\n\t%s\n\t^\n", t->row, t->col, t->value);
								goto error;
							}
						}

						++(*i);

						statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
						statement->token = CopyToken(comma);
						statement->left = (TREE *)calloc(sizeof(TREE), 1);
						statement = statement->left;
						while ((t = (TOKEN *)ArrayGet(tokens, *i))->type != TOKEN_CLOSE_PAREN) {
							statement->token = CopyToken(comma);
							statement->left = CreateTree(tokens, i, false);

							t = (TOKEN *)ArrayGet(tokens, *i);
							if (t->type == TOKEN_COMMA) {
								statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
								++(*i);
								continue;
							} else if (t->type != TOKEN_CLOSE_PAREN) {
								printf("error %d:%d: unexpected token (expected ',' or ')')\n\t%s\n\t^\n", t->row, t->col, t->value);
								goto error;
							}
						}

						TREE *block = expression.right->right;
						block->token = CopyToken(comma);

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_BRACE) {
							printf("error %d:%d: unexpected token (expected '{')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_CLOSE_BRACE) {
							TREE *statement = block;
							for (;;) {
								statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
								statement->token = CopyToken(comma);
								statement->left = CreateTree(tokens, i, false);

								t = (TOKEN *)ArrayGet(tokens, *i);
								if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
									if (!statement->left) {
										break;
									}

									t = (TOKEN *)ArrayGet(tokens, ++(*i));
									if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
										break;
									}
									continue;
								}

								if (!statement->left) {
									goto error;
								}

								break;
							}
						}

						FreeToken(comma);
						ArrayPush(&expression_stack, &expression);

						goto leave;
					}
					case TOKEN_WHILE: {
						if (*i + 4 >= tokens->length) {
							puts("error: unexpected end of input");
							goto error;
						}

						TOKEN *while_ = token;

						TOKEN *t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_OPEN_PAREN) {
							printf("error %d:%d: unexpected token (expected '(')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type == TOKEN_CLOSE_PAREN) {
							printf("error %d:%d: expected an expression\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						TREE expression;
						expression.token = CopyToken(while_);
						expression.left = CreateTree(tokens, i, false);
						expression.right = 0;

						t = (TOKEN *)ArrayGet(tokens, *i);
						if (t->type != TOKEN_CLOSE_PAREN) {
							printf("error %d:%d: unexpected token (expected ')')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						TOKEN *comma = (TOKEN *)calloc(sizeof(TOKEN), 1);
						comma->value = _strdup(",");
						comma->value_length = 1;
						comma->class_ = TOKEN_CLASS_COMMA;
						comma->type = TOKEN_COMMA;

						t = (TOKEN *)ArrayGet(tokens, ++(*i));
						if (t->type != TOKEN_CLOSE_BRACE) {
							TREE *statement = &expression;
							for (;;) {
								statement = (statement->right = (TREE *)calloc(sizeof(TREE), 1));
								statement->token = CopyToken(comma);
								statement->left = CreateTree(tokens, i, false);

								t = (TOKEN *)ArrayGet(tokens, *i);
								if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
									if (!statement->left) {
										break;
									}

									t = (TOKEN *)ArrayGet(tokens, ++(*i));
									if (t->type == TOKEN_CLOSE_BRACE || t->type == TOKEN_SEMICOLON) {
										break;
									}
									continue;
								}

								if (!statement->left) {
									goto error;
								}

								break;
							}
						}

						ArrayPush(&expression_stack, &expression);

						goto leave;
					}
					case TOKEN_BREAK: {
						if (*i + 1 >= tokens->length) {
							puts("error: unexpected end of input");
							goto error;
						}

						TREE e = { 0 };
						e.token = CopyToken(token);
						e.left = e.right = 0;

						TOKEN *t = (TOKEN *)ArrayGet(tokens, *i + 1);
						if (t->type != TOKEN_SEMICOLON) {
							printf("error %d:%d: unexpected token (expected ';')\n\t%s\n\t^\n", t->row, t->col, t->value);
							goto error;
						}

						ArrayPush(&expression_stack, &e);

						break;
					}
					case TOKEN_RETURN: {
						if (*i + 1 >= tokens->length) {
							puts("error: unexpected end of input");
							goto error;
						}

						TREE e;
						e.token = CopyToken(token);
						e.left = e.right = 0;

						TOKEN *t = (TOKEN *)ArrayGet(tokens, *i + 1);
						if (t->type != TOKEN_SEMICOLON) {
							++(*i);
							e.left = CreateTree(tokens, i, false);

							t = (TOKEN *)ArrayGet(tokens, *i);
							if (t->type != TOKEN_SEMICOLON) {
								printf("error %d:%d: unexpected token (expected ';')\n\t%s\n\t^\n", t->row, t->col, t->value);
								goto error;
							}
						}

						ArrayPush(&expression_stack, &e);

						goto leave;
					}
				}

				break;
			}
			case TOKEN_CLASS_ACCESSOR: {
				TREE *e1 = (TREE *)malloc(sizeof(TREE));
				ArrayPop(&expression_stack, e1);

				TREE e = { 0 };
				e.token = CopyToken(token);
				e.left = e1;
				++(*i);
				e.right = CreateTree(tokens, i, false);
				--(*i);
				if (!e.right) {
					goto error;
				}
				
				ArrayPush(&expression_stack, &e);
				break;
			}
		}

		token_previous = token;
	}

leave:
	while (operator_stack.length > 0) {
		TOKEN *o = (*(TOKEN **)ArrayGet(&operator_stack, operator_stack.length - 1));
		if (o->class_ == TOKEN_CLASS_BINARY && expression_stack.length > 1) {
			TREE *e1 = (TREE *)malloc(sizeof(TREE));
			TREE *e2 = (TREE *)malloc(sizeof(TREE));

			ArrayPop(&expression_stack, e2);
			ArrayPop(&expression_stack, e1);

			TREE expression;
			ArrayPop(&operator_stack, &expression.token);
			expression.token = CopyToken(expression.token);
			expression.left = e1;
			expression.right = e2;

			ArrayPush(&expression_stack, &expression);
		} else {
			TREE *e = (TREE *)malloc(sizeof(TREE));

			ArrayPop(&expression_stack, e);

			TREE expression;
			ArrayPop(&operator_stack, &expression.token);
			expression.token = CopyToken(expression.token);
			expression.left = e;
			expression.right = 0;

			ArrayPush(&expression_stack, &expression);
		}
	}

	ArrayFree(&operator_stack);
	if (expression_stack.length > 0) {
		TREE *ret = (TREE *)malloc(sizeof(TREE));
		ArrayPop(&expression_stack, ret);
		ArrayFree(&expression_stack);
		return ret;
	} else {
		ArrayFree(&expression_stack);
		return 0;
	}

error:
	ArrayFree(&operator_stack);
	ArrayFree(&expression_stack);
	return 0;
}

void PrintTree(TREE *tree) {
	_PrintTree(tree, L"", 1);
}

TREE *CopyTree(TREE *tree) {
	TREE *copy = (TREE *)calloc(sizeof(TREE), 1);

	if (tree && tree->token) {
		copy->token = (TOKEN *)malloc(sizeof(TOKEN));
		memcpy(copy->token, tree->token, sizeof(TOKEN));
		copy->token->value = _strdup(tree->token->value);

		if (tree->left) {
			copy->left = CopyTree(tree->left);
		}
		if (tree->right) {
			copy->right = CopyTree(tree->right);
		}
	}

	return copy;
}

void FreeTree(TREE *tree) {
	if (tree) {
		if (tree->left) {
			FreeTree(tree->left);
		}
		if (tree->right) {
			FreeTree(tree->right);
		}

		if (tree->token) {
			if (tree->token->value) {
				free(tree->token->value);
			}
			free(tree->token);
		}

		free(tree);
	}
}

void _PrintTree(TREE *tree, wchar_t *prefix, int tail) {
	if (!tree || !tree->token || !tree->token->value) {
		return;
	}

	wchar_t buffer[0xFFF];
	buffer[MultiByteToWideChar(CP_ACP, MB_PRECOMPOSED, tree->token->value, tree->token->value_length, buffer, tree->token->value_length)] = 0;

	_setmode(_fileno(stdout), _O_U16TEXT);
	wprintf(L"%ls%ls%ls\n", prefix, tail ? L"└── " : L"├── ", buffer);
	_setmode(_fileno(stdout), _O_TEXT);

	if (tree->left) {
		wsprintf(buffer, L"%ws%ws", prefix, tail ? L"    " : L"│   ");

		_PrintTree(tree->left, buffer, tree->right ? 0 : 1);
	}

	if (tree->right) {
		wsprintf(buffer, L"%ws%ws", prefix, tail ? L"    " : L"│   ");

		_PrintTree(tree->right, buffer, 1);
	}
}