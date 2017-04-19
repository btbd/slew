#pragma once

enum {
	VALUE_NUMBER = 0,
	VALUE_STRING,
	VALUE_ARRAY,
	VALUE_FUNCTION,
	VALUE_COMPILED_FUNCTION,
	VALUE_BREAK,
	VALUE_SCOPE
};

#define RETURN_OUT(v) v.return_ = 1; return v;
#define COMPILED_FUNC VALUE (*)(ARRAY *arguments)

typedef struct {
	unsigned int type;
	char *name;
	int stack_id, return_;

	union {
		double number;
		struct {
			char *string;
			unsigned int string_length;
		};
		struct {
			TREE *function;
			unsigned int stack_last;
		};
		void *compiled_function;
		ARRAY *array;
	};

	ARRAY *properties;
} VALUE;

void SetupLibraries();
VALUE Eval(TREE *tree, int stack_id);
int Stack_InStack(int stack_id, char *name);
VALUE *Stack_Get(char *name);
VALUE *Stack_GetWithProperties(TREE *var);
VALUE *Stack_Set(TREE *var, VALUE *value, int stack_id);
VALUE *Stack_Push(char *name, VALUE *value, int stack_id);
void PrintStack();
void Value_Free(VALUE *value);
VALUE Value_Number(double number);
VALUE Value_String(char *string_);
VALUE Value_Array();
VALUE Value_Function(TREE *function);
VALUE Value_Compiled_Function(void *function);
VALUE Value_Scope();
void Value_MergeProperties(VALUE *value, ARRAY *properties);
VALUE Value_Copy(VALUE *value);
VALUE Value_CopyWithName(VALUE *value, char *name);
VALUE *Value_SetProperty(VALUE *value, char *name, VALUE *property);
VALUE *Value_GetProperty(VALUE *value, char *name);
int Value_HasProperty(VALUE *value, char *name);
void Value_CallGetter(VALUE *value);
void Value_CallSetter(VALUE *value);
void Value_Call(VALUE *value, VALUE args);
void PrintValue(VALUE *value);