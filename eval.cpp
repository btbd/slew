#include "main.h"

ARRAY *stack = Array_New(sizeof(VALUE));

VALUE Global_get(ARRAY *arguments) {
	if (arguments->length > 1) {
		VALUE *value = (VALUE *)Array_Get(arguments, 0);
		if (value->type == VALUE_ARRAY) {
			return Value_Copy((VALUE *)Array_Get(value->array, (int)((VALUE *)Array_Get(arguments, 1))->number));
		} else if (value->type == VALUE_STRING) {
			char s[2];
			*s = value->string[(int)((VALUE *)Array_Get(arguments, 1))->number];
			s[1] = 0;
			return Value_String(s);
		}
	} else if (arguments->length > 0) {
		return Value_Copy((VALUE *)Array_Get(arguments, 0));
	}

	return Value_Number(0);
}

VALUE Global_set(ARRAY *arguments) {
	if (arguments->length > 2 && ((VALUE *)Array_Get(arguments, 1))->type == VALUE_NUMBER) {
		VALUE *value = (VALUE *)Array_Get(arguments, 0);
		int index = (int)((VALUE *)Array_Get(arguments, 1))->number;
		VALUE *set = (VALUE *)Array_Get(arguments, 2);
		if (value->type == VALUE_ARRAY && index < (int)value->array->length) {
			Array_Set(value->array, index, &Value_Copy(set));
			return Value_Copy(value);
		} else if (value->type == VALUE_STRING && set->type == VALUE_STRING && index < (int)value->string_length) {
			value->string[index] = *set->string;
			value->string_length = strlen(value->string);
			return Value_Copy(value);
		}
	}

	return Value_Number(0);
}

VALUE Global_length(ARRAY *arguments) {
	static char temp[0xFFF];

	if (arguments->length > 0) {
		VALUE *v = (VALUE *)Array_Get(arguments, 0);
		switch (v->type) {
			case VALUE_NUMBER:
				return Value_Number(sprintf(temp, "%g", v->number));
			case VALUE_STRING:
				return Value_Number(v->string_length);
			case VALUE_ARRAY:
				return Value_Number(v->array ? v->array->length : 0);
		}
	}

	return Value_Number(0);
}

// Console
VALUE Console_print(ARRAY *arguments) {
	if (arguments->length > 0) PrintValue((VALUE *)Array_Get(arguments, 0));
	return Value_Number(0);
}

VALUE Console_println(ARRAY *arguments) {
	if (arguments->length > 0) PrintValue((VALUE *)Array_Get(arguments, 0));
	putchar('\n');
	return Value_Number(0);
}

VALUE Console_clear(ARRAY *arguments) {
	system("cls");
	return Value_Number(0);
}

// Math
VALUE Math_abs(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(fabs(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_acos(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(acos(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_acosh(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(acosh(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_asin(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(asin(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_asinh(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(asinh(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_atan(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(atan(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_atanh(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(atanh(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_atan2(ARRAY *arguments) {
	if (arguments->length > 1) {
		return Value_Number(atan2(((VALUE *)Array_Get(arguments, 0))->number, ((VALUE *)Array_Get(arguments, 1))->number));
	}
	return Value_Number(0);
}

VALUE Math_cbrt(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(cbrt(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_ceil(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(ceil(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_cos(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(cos(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_cosh(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(cosh(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_deg(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(((VALUE *)Array_Get(arguments, 0))->number * (double)180 / (double)3.141592653589793);
	}
	return Value_Number(0);
}

VALUE Math_exp(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(exp(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_expm1(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(expm1(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_floor(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(floor(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_log(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(log(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_log1p(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(log1p(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_log10(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(log10(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_log2(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(log2(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_pow(ARRAY *arguments) {
	if (arguments->length > 1) {
		return Value_Number(pow(((VALUE *)Array_Get(arguments, 0))->number, ((VALUE *)Array_Get(arguments, 1))->number));
	}
	return Value_Number(0);
}

VALUE Math_rad(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(((VALUE *)Array_Get(arguments, 0))->number * (double)3.141592653589793 / (double)180);
	}
	return Value_Number(0);
}

VALUE Math_random(ARRAY *arguments) {
	return Value_Number((((double)(rand() % 0xFF) / (double)0xFF) + ((double)(rand() % 0xFF) / (double)0xFF)) / 2);
}

VALUE Math_round(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(round(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_sign(ARRAY *arguments) {
	if (arguments->length > 0) {
		double v = ((VALUE *)Array_Get(arguments, 0))->number;
		return Value_Number(v == 0 ? 0 : v < 0 ? -1 : 1);
	}
	return Value_Number(0);
}

VALUE Math_sin(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(sin(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_sinh(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(sinh(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_sqrt(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(sqrt(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_tan(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(tan(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_tanh(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(tanh(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

VALUE Math_trunc(ARRAY *arguments) {
	if (arguments->length > 0) {
		return Value_Number(trunc(((VALUE *)Array_Get(arguments, 0))->number));
	}
	return Value_Number(0);
}

void SetupLibraries() {
	Stack_Push("get", &Value_Compiled_Function(&Global_get), 0);
	Stack_Push("set", &Value_Compiled_Function(&Global_set), 0);
	Stack_Push("length", &Value_Compiled_Function(&Global_length), 0);

	VALUE *Console = Stack_Push("Console", &Value_Number(0), 0);
	Value_SetProperty(Console, "print", &Value_Compiled_Function(&Console_print));
	Value_SetProperty(Console, "println", &Value_Compiled_Function(&Console_println));
	Value_SetProperty(Console, "clear", &Value_Compiled_Function(&Console_clear));

	VALUE *Math = Stack_Push("Math", &Value_Number(0), 0);
	srand(GetTickCount());
	Value_SetProperty(Math, "E", &Value_Number(2.718281828459045));
	Value_SetProperty(Math, "LN2", &Value_Number(0.6931471805599453));
	Value_SetProperty(Math, "LN10", &Value_Number(2.302585092994046));
	Value_SetProperty(Math, "PI", &Value_Number(3.141592653589793));
	Value_SetProperty(Math, "SQRT1_2", &Value_Number(0.7071067811865476));
	Value_SetProperty(Math, "SQRT2", &Value_Number(1.4142135623730951));
	Value_SetProperty(Math, "abs", &Value_Compiled_Function(&Math_abs));
	Value_SetProperty(Math, "acos", &Value_Compiled_Function(&Math_acos));
	Value_SetProperty(Math, "acosh", &Value_Compiled_Function(&Math_acosh));
	Value_SetProperty(Math, "asin", &Value_Compiled_Function(&Math_asin));
	Value_SetProperty(Math, "asinh", &Value_Compiled_Function(&Math_asinh));
	Value_SetProperty(Math, "atan", &Value_Compiled_Function(&Math_atan));
	Value_SetProperty(Math, "atanh", &Value_Compiled_Function(&Math_atanh));
	Value_SetProperty(Math, "atan2", &Value_Compiled_Function(&Math_atan2));
	Value_SetProperty(Math, "cbrt", &Value_Compiled_Function(&Math_cbrt));
	Value_SetProperty(Math, "ceil", &Value_Compiled_Function(&Math_ceil));
	Value_SetProperty(Math, "cos", &Value_Compiled_Function(&Math_cos));
	Value_SetProperty(Math, "cosh", &Value_Compiled_Function(&Math_cosh));
	Value_SetProperty(Math, "deg", &Value_Compiled_Function(&Math_deg));
	Value_SetProperty(Math, "exp", &Value_Compiled_Function(&Math_exp));
	Value_SetProperty(Math, "expm1", &Value_Compiled_Function(&Math_expm1));
	Value_SetProperty(Math, "floor", &Value_Compiled_Function(&Math_floor));
	Value_SetProperty(Math, "log", &Value_Compiled_Function(&Math_log));
	Value_SetProperty(Math, "log1p", &Value_Compiled_Function(&Math_log1p));
	Value_SetProperty(Math, "log10", &Value_Compiled_Function(&Math_log10));
	Value_SetProperty(Math, "pow", &Value_Compiled_Function(&Math_pow));
	Value_SetProperty(Math, "rad", &Value_Compiled_Function(&Math_rad));
	Value_SetProperty(Math, "random", &Value_Compiled_Function(&Math_random));
	Value_SetProperty(Math, "round", &Value_Compiled_Function(&Math_round));
	Value_SetProperty(Math, "sign", &Value_Compiled_Function(&Math_sign));
	Value_SetProperty(Math, "sin", &Value_Compiled_Function(&Math_sin));
	Value_SetProperty(Math, "sinh", &Value_Compiled_Function(&Math_sinh));
	Value_SetProperty(Math, "sqrt", &Value_Compiled_Function(&Math_sqrt));
	Value_SetProperty(Math, "tan", &Value_Compiled_Function(&Math_tan));
	Value_SetProperty(Math, "tanh", &Value_Compiled_Function(&Math_tanh));
	Value_SetProperty(Math, "trunc", &Value_Compiled_Function(&Math_trunc));
}

VALUE Eval(TREE *tree, int stack_id) {
	static char temp[0xFFF];

	if (!tree) {
		return { 0 };
	}

	__try {
		switch (tree->token->type) {
			case TOKEN_DECLARATION: {
				VALUE right = Eval(tree->right, stack_id);

				if (Stack_InStack(stack_id, tree->left->token->value)) {
					Stack_Set(tree->left, &right, stack_id);
				} else {
					VALUE n = { 0 };
					Stack_Push(tree->left->token->value, &n, stack_id);
					Stack_Set(tree->left, &right, stack_id);
				}

				return right;
			}
			case TOKEN_EQUAL: {
				VALUE right = Eval(tree->right, stack_id);

				Stack_Set(tree->left, &right, stack_id);

				return right;
			}
			case TOKEN_WORD: {
				VALUE *v = Stack_GetWithProperties(tree);
				if (v) {
					return Value_Copy(v);
				}

				break;
			}
			case TOKEN_CALL: {
				VALUE func = Value_Copy(Stack_GetWithProperties(tree->left));
				if (func.type != VALUE_FUNCTION && func.type != VALUE_COMPILED_FUNCTION) {
					break;
				}

				if (func.type == VALUE_FUNCTION) {
					VALUE args = Value_Array();
					TREE *tree_base = tree->left;
					TREE *this_ = tree_base;
					TREE *name = func.function->left;
					while (tree->right && tree->right->left) {
						VALUE v = Eval(tree->right->left, stack_id);
						if (name) {
							v.name = name->token->value;
							name = name->left;
						} else {
							v.name = 0;
						}
						tree = tree->right;
						Array_Push(args.array, &v);
					}

					while (this_->left && this_->left->left) {
						this_ = this_->left;
					}

					++stack_id;
					Stack_Push("", &Value_Scope(), -1);
					Stack_Push("arguments", &args, stack_id);
					TREE *copy = this_->left; this_->left = 0;
					Stack_Push("this", Stack_GetWithProperties(tree_base), stack_id);
					this_->left = copy;

					for (unsigned int i = 0; i < args.array->length; ++i) {
						VALUE *v = (VALUE *)Array_Get(args.array, i);
						if (!v->name) {
							break;
						}
						Stack_Push(v->name, v, stack_id);
					}

					tree = func.function;
					VALUE v = { 0 };
					while (tree->right) {
						v = Eval(tree->right->left, stack_id);
						if (v.return_) {
							break;
						}
						Value_Free(&v);
						v = { 0 };
						tree = tree->right;
					}

					while (stack->length > 0) {
						VALUE v;
						Array_Pop(stack, &v);
						if (v.type == VALUE_SCOPE) {
							Value_Free(&v);
							break;
						}
						Value_Free(&v);
					}

					Value_Free(&func);
					Value_Free(&args);
					return v;
				} else {
					VALUE args = Value_Array();
					while (tree->right && tree->right->left) {
						VALUE v = Eval(tree->right->left, stack_id);
						tree = tree->right;
						Array_Push(args.array, &v);
					}

					VALUE r = ((COMPILED_FUNC)func.compiled_function)(args.array);
					Value_Free(&func);
					Value_Free(&args);
					return r;
				}

				break;
			}
			case TOKEN_IF: {
				if (Eval(tree->left->left, ++stack_id).number) {
					tree = tree->left;
					while (tree->right) {
						VALUE v = Eval(tree->right->left, stack_id);
						if (v.return_) {
							RETURN_OUT(v);
						}
						tree = tree->right;
					}
				}

				while (tree->right) {
					if (tree->right->token->type == TOKEN_ELSE_IF) {
						TREE *t = tree->right->left;
						if (Eval(t->left, ++stack_id).number) {
							while (t->right) {
								VALUE v = Eval(t->right->left, stack_id);
								if (v.return_) {
									RETURN_OUT(v);
								}
								t = t->right;
							}
						}
					} else {
						++stack_id;
						TREE *t = tree->right;
						while (t->right) {
							VALUE v = Eval(t->right->left, stack_id);
							if (v.return_) {
								RETURN_OUT(v);
							}
							t = t->right;
						}
					}

					tree = tree->right;
				}

				break;
			}
			case TOKEN_WHILE: {
				TREE *t;
				++stack_id;
				char break_ = 0;
				while (!break_ && Eval(tree->left, stack_id).number) {
					t = tree;
					while (t->right) {
						VALUE v = Eval(t->right->left, stack_id);
						if (v.return_) {
							RETURN_OUT(v);
						} else if (v.type == VALUE_BREAK) {
							break_ = 1;
							break;
						}
						t = t->right;
					}
				}

				break;
			}
			case TOKEN_FOR: {
				++stack_id;
				if (tree->left->left) {
					Eval(tree->left->left, stack_id);
					TREE *t = tree->left;
					while (t->right) {
						Eval(t->right->left, stack_id);
						t = t->right;
					}
				}

				char break_ = 0;
				while (!tree->right->left || Eval(tree->right->left, stack_id).number) {
					TREE *t = tree->right->right;
					while (t->right) {
						VALUE v = Eval(t->right->left, stack_id);
						if (v.return_) {
							RETURN_OUT(v);
						} else if (v.type == VALUE_BREAK) {
							break_ = 1;
							break;
						}
						t = t->right;
					}

					if (break_) {
						break;
					}

					t = tree->right->right;
					if (t->left->left) {
						Eval(t->left->left, stack_id);
						t = tree->right->right->left;
						while (t->right) {
							Eval(t->right->left, stack_id);
							t = t->right;
						}
					}
				}

				break;
			}
			case TOKEN_QUESTION: {
				if (Eval(tree->left, stack_id).number) {
					return Eval(tree->right->left, stack_id);
				} else {
					return Eval(tree->right->right, stack_id);
				}
			}
			case TOKEN_PLUS_PLUS: {
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v && v->type == VALUE_NUMBER) {
					++(v->number);
					Value_CallSetter(v);

					VALUE r = { 0 };
					r.type = VALUE_NUMBER;
					r.number = v->number;
					if (*tree->token->value != '+') {
						--r.number;
					}
					return r;
				}

				break;
			}
			case TOKEN_MINUS_MINUS: {
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v && v->type == VALUE_NUMBER) {
					--(v->number);
					Value_CallSetter(v);

					VALUE r = { 0 };
					r.type = VALUE_NUMBER;
					r.number = v->number;
					if (*tree->token->value != '-') {
						++r.number;
					}
					return r;
				}

				break;
			}
			case TOKEN_PLUS_EQUAL: {
				tree->token->type = TOKEN_PLUS;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_MINUS_EQUAL: {
				tree->token->type = TOKEN_MINUS;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_MULTIPLY_EQUAL: {
				tree->token->type = TOKEN_MULTIPLY;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_DIVIDE_EQUAL: {
				tree->token->type = TOKEN_DIVIDE;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_AND_EQUAL: {
				tree->token->type = TOKEN_AND;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_OR_EQUAL: {
				tree->token->type = TOKEN_OR;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_XOR_EQUAL: {
				tree->token->type = TOKEN_XOR;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_SHIFT_LEFT_EQUAL: {
				tree->token->type = TOKEN_SHIFT_LEFT;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_SHIFT_RIGHT_EQUAL: {
				tree->token->type = TOKEN_SHIFT_RIGHT;
				VALUE n = Eval(tree, stack_id);
				VALUE *v = Stack_GetWithProperties(tree->left);
				if (v) {
					Stack_Set(tree->left, &n, stack_id);
					return n;
				}
				break;
			}
			case TOKEN_PLUS: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE value = { 0 };
				if (left.type == VALUE_NUMBER && right.type == VALUE_NUMBER) {
					value.type = VALUE_NUMBER;
					value.number = left.number + right.number;
				} else if (left.type == VALUE_STRING && right.type == VALUE_STRING) {
					value.type = VALUE_STRING;
					value.string_length = left.string_length + right.string_length;
					value.string = (char *)calloc(value.string_length + 1, 1);

					sprintf(value.string, "%s%s", left.string, right.string);
				} else if (left.type == VALUE_STRING && right.type == VALUE_NUMBER) {
					value.type = VALUE_STRING;

					sprintf(temp, "%g", right.number);

					value.string_length = left.string_length + strlen(temp);
					value.string = (char *)calloc(value.string_length + 1, 1);

					sprintf(value.string, "%s%s", left.string, temp);
				} else if (left.type == VALUE_NUMBER && right.type == VALUE_STRING) {
					value.type = VALUE_STRING;

					sprintf(temp, "%g", left.number);

					value.string_length = right.string_length + strlen(temp);
					value.string = (char *)calloc(value.string_length + 1, 1);

					sprintf(value.string, "%s%s", temp, right.string);
				} else {
					value.type = VALUE_NUMBER;
					value.number = 0;
				}

				Value_Free(&left);
				Value_Free(&right);
				return value;
			}
			case TOKEN_MINUS: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? left.number - right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_MULTIPLY: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? left.number * right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_DIVIDE: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? left.number / right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_AND: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? (int)left.number & (int)right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_OR: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? (int)left.number | (int)right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_XOR: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? (int)left.number ^ (int)right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_SHIFT_LEFT: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? (int)left.number << (int)right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_SHIFT_RIGHT: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.type == VALUE_NUMBER && right.type == VALUE_NUMBER ? (int)left.number >> (int)right.number : 0;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_EQUAL_EQUAL: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				if (left.type == VALUE_STRING && right.type == VALUE_STRING) {
					v.number = strcmp(left.string, right.string) == 0 ? 1 : 0;
				} else {
					v.number = left.number == right.number;
				}

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_NOT_EQUAL: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				if (left.type == VALUE_STRING && right.type == VALUE_STRING) {
					v.number = strcmp(left.string, right.string) == 0 ? 0 : 1;
				} else {
					v.number = left.number == right.number;
				}

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_AND_AND: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.number && right.number;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_OR_OR: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.number || right.number;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_GREATER: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.number > right.number;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_GREATER_EQUAL: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.number >= right.number;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_LESS: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.number < right.number;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_LESS_EQUAL: {
				VALUE left = Eval(tree->left, stack_id);
				VALUE right = Eval(tree->right, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = left.number <= right.number;

				Value_Free(&left);
				Value_Free(&right);
				return v;
			}
			case TOKEN_NOT: {
				VALUE left = Eval(tree->left, stack_id);

				VALUE v = { 0 };
				v.type = VALUE_NUMBER;
				v.number = !left.number;

				Value_Free(&left);
				return v;
			}
			case TOKEN_DECIMAL_NUMBER: {
				return Value_Number(atof(tree->token->value));
			}
			case TOKEN_HEX_NUMBER: {
				return Value_Number(strtol(tree->token->value + 2, NULL, 16));
			}
			case TOKEN_BINARY_NUMBER: {
				return Value_Number(strtol(tree->token->value + 2, NULL, 2));
			}
			case TOKEN_STRING: {
				VALUE v = Value_String(tree->token->value + 1);
				*(v.string + --v.string_length) = 0;
				return v;
			}
			case TOKEN_ARRAY: {
				VALUE v = Value_Array();

				TREE *e = tree;
				if (e->left) {
					Array_Push(v.array, &Eval(e->left, stack_id));

					while (e->right) {
						e = e->right;

						Array_Push(v.array, &Eval(e->left, stack_id));
					}
				}

				return v;
			}
			case TOKEN_FUNC: {
				return Value_Function(tree);
			}
			case TOKEN_BREAK: {
				VALUE v = { 0 };
				v.type = VALUE_BREAK;
				return v;
			}
			case TOKEN_RETURN: {
				VALUE v = Eval(tree->left, stack_id);
				v.return_ = 1;
				return v;
			}
		}
	} __except (EXCEPTION_EXECUTE_HANDLER) {
		printf("error: stack overflow\n");
		exit(EXIT_FAILURE);
	}

	return { 0 };
}

int Stack_InStack(int stack_id, char *name) {
	for (unsigned int i = 0; i < stack->length; ++i) {
		VALUE *v = (VALUE *)Array_Get(stack, i);
		if (v->stack_id == stack_id && strcmp(v->name, name) == 0) {
			return 1;
		}
	}

	return 0;
}

VALUE *Stack_Get(char *name) {
	for (int i = stack->length - 1; i > -1; --i) {
		VALUE *v = (VALUE *)Array_Get(stack, i);
		if (v->type == VALUE_SCOPE) {
			i = v->stack_last + 1;
		} else if (strcmp(v->name, name) == 0) {
			Value_CallGetter(v);

			return v;
		}
	}

	return 0;
}

VALUE *Stack_GetWithProperties(TREE *var) {
	VALUE *v = 0;
	for (int i = stack->length - 1; i > -1; --i) {
		VALUE *n = (VALUE *)Array_Get(stack, i);
		if (n->type == VALUE_SCOPE) {
			i = n->stack_last + 1;
		} else if (strcmp(n->name, var->token->value) == 0) {
			v = n;
			break;
		}
	}

	if (v) {
		while (var->left) {
			v = Value_GetProperty(v, var->left->token->value);
			var = var->left;
		}

		Value_CallGetter(v);

		return v;
	}

	return 0;
}

VALUE *Stack_Set(TREE *var, VALUE *value, int stack_id) {
	VALUE *v = 0;
	for (int i = stack->length - 1; i > -1; --i) {
		VALUE *n = (VALUE *)Array_Get(stack, i);
		if (n->type == VALUE_SCOPE) {
			i = n->stack_last + 1;
		} else if (strcmp(n->name, var->token->value) == 0) {
			v = n;
			break;
		}
	}

	if (!v) {
		VALUE n = { 0 };
		v = Stack_Push(var->token->value, &n, stack_id);
	}

	char *name = var->token->value;
	while (var->left) {
		v = Value_GetProperty(v, var->left->token->value);
		name = var->left->token->value;
		var = var->left;
	}

	Value_Free(v);
	ARRAY *properties = v->properties;
	*v = Value_CopyWithName(value, name);
	if (properties) {
		Value_MergeProperties(v, properties);
		Array_Free(properties);
	}
	
	Value_CallSetter(v);

	return v;
}

VALUE *Stack_Push(char *name, VALUE *value, int stack_id) {
	VALUE copy = Value_CopyWithName(value, name);
	copy.stack_id = stack_id;
	return (VALUE *)Array_Push(stack, &copy);
}

void PrintStack() {
	for (int i = stack->length - 1; i > -1; --i) {
		VALUE *v = (VALUE *)Array_Get(stack, i);
		printf("%s - ", v->name ? v->name : "");
		PrintValue(v);
		puts("");
	}
}

void Value_Free(VALUE *value) {
	switch (value->type) {
		case VALUE_STRING:
			if (value->string) {
				free(value->string);
			}
			break;
		case VALUE_ARRAY: {
			if (value->array) {
				for (unsigned int i = 0; i < value->array->length; ++i) {
					Value_Free((VALUE *)Array_Get(value->array, i));
				}
				Array_Free(value->array);
			}
			break;
		}
		case VALUE_FUNCTION: 
			FreeTree(value->function);
			free(value->function);
			break;
	}
}

VALUE Value_Number(double number) {
	VALUE value = { 0 };

	value.type = VALUE_NUMBER;
	value.number = number;

	return value;
}

VALUE Value_String(char *string_) {
	VALUE value = { 0 };

	value.type = VALUE_STRING;
	value.string_length = strlen(string_);
	value.string = (char *)calloc(value.string_length + 1, 1);
	memcpy(value.string, string_, value.string_length);

	char *s = value.string;
	for (int l = value.string_length; *s && l > 1; ++s, --l) {
		if (memcmp(s, "\\'", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\'';
		} else if (memcmp(s, "\\\"", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\"';
		} else if (memcmp(s, "\\\\", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\\';
		} else if (memcmp(s, "\\n", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\n';
		} else if (memcmp(s, "\\r", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\r';
		} else if (memcmp(s, "\\t", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\t';
		} else if (memcmp(s, "\\b", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\b';
		} else if (memcmp(s, "\\f", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\f';
		} else if (memcmp(s, "\\v", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\v';
		} else if (memcmp(s, "\\0", 2) == 0) {
			--value.string_length;
			memmove(s, s + 1, l);
			*s = '\0';
			value.string_length = strlen(value.string) + 1;
			break;
		} else if (l > 3 && memcmp(s, "\\x", 2) == 0) {
			value.string_length -= 3;
			unsigned char v = (unsigned char)strtol(s + 2, NULL, 16);
			memmove(s, s + 3, l);
			*s = v;
		}
	}

	return value;
}

VALUE Value_Array() {
	VALUE value = { 0 };

	value.type = VALUE_ARRAY;
	value.array = Array_New(sizeof(VALUE));

	return value;
}

VALUE Value_Function(TREE *function) {
	VALUE value = { 0 };

	value.type = VALUE_FUNCTION;
	value.function = CopyTree(function);
	value.stack_last = stack->length;

	return value;
}

VALUE Value_Compiled_Function(void *function) {
	VALUE value = { 0 };

	value.type = VALUE_COMPILED_FUNCTION;
	value.compiled_function = function;

	return value;
}

VALUE Value_Scope() {
	VALUE value = { 0 };

	value.type = VALUE_SCOPE;
	for (int i = stack->length - 1; i > -1; --i) {
		VALUE *v = (VALUE *)Array_Get(stack, i);
		if (v->stack_id == 0) {
			value.stack_last = i;
			break;
		}
	}

	return value;
}

void Value_MergeProperties(VALUE *value, ARRAY *properties) {
	if (properties) {
		if (!value->properties) {
			value->properties = Array_New(sizeof(VALUE));
		}
		for (unsigned int i = 0; i < properties->length; ++i) {
			VALUE *p = (VALUE *)Array_Get(properties, i);
			Array_Push(value->properties, &Value_CopyWithName(p, p->name));
		}
	}
}

VALUE Value_Copy(VALUE *value) {
	VALUE v = { 0 };

	if (value) {
		v.type = value->type;
		switch (value->type) {
			case VALUE_NUMBER:
				v.number = value->number;
				break;
			case VALUE_STRING:
				if (value->string) {
					v.string = (char *)calloc(value->string_length + 1, 1);
					memcpy(v.string, value->string, value->string_length);
					v.string_length = value->string_length;
				}
				break;
			case VALUE_ARRAY:
				if (value->array) {
					v.array = Array_New(sizeof(VALUE));
					for (unsigned int i = 0; i < value->array->length; ++i) {
						Array_Push(v.array, &Value_Copy((VALUE *)Array_Get(value->array, i)));
					}
				}
				break;
			case VALUE_FUNCTION:
				v.stack_last = value->stack_last;
				v.function = CopyTree(value->function);
				break;
			case VALUE_COMPILED_FUNCTION: 
				v.compiled_function = value->compiled_function;
				break;
			case VALUE_SCOPE:
				v.stack_last = value->stack_last;
		}

		if (value->properties) {
			v.properties = Array_New(sizeof(VALUE));
			for (unsigned int i = 0; i < value->properties->length; ++i) {
				VALUE *p = (VALUE *)Array_Get(value->properties, i);
				Array_Push(v.properties, &Value_CopyWithName(p, p->name));
			}
		}
	}

	return v;
}

VALUE Value_CopyWithName(VALUE *value, char *name) {
	VALUE copy = Value_Copy(value);
	copy.name = _strdup(name);
	return copy;
}

VALUE *Value_SetProperty(VALUE *value, char *name, VALUE *property) {
	if (!value->properties) {
		value->properties = Array_New(sizeof(VALUE));
		return (VALUE *)Array_Push(value->properties, &Value_CopyWithName(property, name));
	} else {
		for (unsigned int i = 0; i < value->properties->length; ++i) {
			VALUE *v = (VALUE *)Array_Get(value->properties, i);
			if (strcmp(v->name, name) == 0) {
				Value_Free(v);
				*v = Value_CopyWithName(property, name);
				return v;
			}
		}

		return (VALUE *)Array_Push(value->properties, &Value_CopyWithName(property, name));
	}
}

VALUE *Value_GetProperty(VALUE *value, char *name) {
	if (value->properties) {
		for (unsigned int i = 0; i < value->properties->length; ++i) {
			VALUE *v = (VALUE *)Array_Get(value->properties, i);
			if (strcmp(v->name, name) == 0) {
				return v;
			}
		}
	} else {
		value->properties = Array_New(sizeof(VALUE));
	}

	VALUE v = { 0 };
	return (VALUE *)Array_Push(value->properties, &Value_CopyWithName(&v, name));
}

int Value_HasProperty(VALUE *value, char *name) {
	if (value->properties) {
		for (unsigned int i = 0; i < value->properties->length; ++i) {
			VALUE *v = (VALUE *)Array_Get(value->properties, i);
			if (strcmp(v->name, name) == 0) {
				return 1;
			}
		}
	}

	return 0;
}

void Value_CallGetter(VALUE *v) {
	if (Value_HasProperty(v, "get")) {
		VALUE *get = Value_GetProperty(v, "get");
		if (get->type == VALUE_FUNCTION && get->function) {
			VALUE args = Value_Array();
			Stack_Push("", &Value_Scope(), -1);
			Stack_Push("arguments", &args, 1);
			Stack_Push("this", v, 1);

			VALUE v = { 0 };
			TREE *tree = get->function;
			while (tree->right) {
				v = Eval(tree->right->left, 1);
				if (v.return_) {
					break;
				}
				Value_Free(&v);
				v = { 0 };
				tree = tree->right;
			}

			while (stack->length > 0) {
				VALUE v;
				Array_Pop(stack, &v);
				if (v.type == VALUE_SCOPE) {
					Value_Free(&v);
					break;
				}
				Value_Free(&v);
			}

			Value_Free(&v);
			Value_Free(&args);
		} else if (get->type == VALUE_COMPILED_FUNCTION && get->compiled_function) {
			VALUE args = Value_Array();
			Value_Free(&((COMPILED_FUNC)get->compiled_function)(args.array));
			Value_Free(&args);
		}
	}
}

void Value_CallSetter(VALUE *v) {
	if (Value_HasProperty(v, "set")) {
		VALUE *set = Value_GetProperty(v, "set");
		if (set->type == VALUE_FUNCTION && set->function) {
			VALUE args = Value_Array();
			Stack_Push("", &Value_Scope(), -1);
			Stack_Push("arguments", &args, 1);
			Stack_Push("this", v, 1);

			VALUE v = { 0 };
			TREE *tree = set->function;
			while (tree->right) {
				v = Eval(tree->right->left, 1);
				if (v.return_) {
					break;
				}
				Value_Free(&v);
				v = { 0 };
				tree = tree->right;
			}

			while (stack->length > 0) {
				VALUE v;
				Array_Pop(stack, &v);
				if (v.type == VALUE_SCOPE) {
					Value_Free(&v);
					break;
				}
				Value_Free(&v);
			}

			Value_Free(&v);
			Value_Free(&args);
		} else if (set->type == VALUE_COMPILED_FUNCTION && set->compiled_function) {
			VALUE args = Value_Array();
			Value_Free(&((COMPILED_FUNC)set->compiled_function)(args.array));
			Value_Free(&args);
		}
	}
}

void Value_Call(VALUE *value, VALUE args) {
	if (value) {
		if (value->type == VALUE_FUNCTION && value->function) {
			Stack_Push("", &Value_Scope(), -1);
			Stack_Push("arguments", &args, 1);
			Stack_Push("this", &Value_Number(0), 1);

			TREE *name = value->function->left;
			for (unsigned int i = 0; i < args.array->length; ++i) {
				if (name) {
					VALUE *v = (VALUE *)Array_Get(args.array, i);
					v->name = name->token->value;
					Stack_Push(name->token->value, v, 1);
					name = name->left;
				} else {
					break;
				}
			}

			VALUE v = { 0 };
			TREE *tree = value->function;
			while (tree->right) {
				v = Eval(tree->right->left, 1);
				if (v.return_) {
					break;
				}
				Value_Free(&v);
				v = { 0 };
				tree = tree->right;
			}

			while (stack->length > 0) {
				VALUE v;
				Array_Pop(stack, &v);
				if (v.type == VALUE_SCOPE) {
					Value_Free(&v);
					break;
				}
				Value_Free(&v);
			}

			Value_Free(&v);
			Value_Free(&args);
		} else if (value->type == VALUE_COMPILED_FUNCTION && value->compiled_function) {
			VALUE args = Value_Array();
			Value_Free(&((COMPILED_FUNC)value->compiled_function)(args.array));
			Value_Free(&args);
		}
	}
}

void PrintValue(VALUE *value) {
	if (!value) {
		return;
	}

	switch (value->type) {
		case VALUE_NUMBER: {
			printf("%.17g", value->number);
			break;
		}
		case VALUE_STRING:
			printf("\"%s\"", value->string);
			break;
		case VALUE_ARRAY: {
			printf("[");
			for (unsigned int i = 0; i < value->array->length; ++i) {
				PrintValue((VALUE *)Array_Get(value->array, i));
				if (i + 1 < value->array->length) {
					printf(", ");
				}
			}
			printf("]");

			break;
		}
		default:
			printf("0x%08x", (unsigned int)value->array);
			break;
	}
}