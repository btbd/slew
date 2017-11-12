#pragma once

#define COMPILED_FUNC VALUE (*)(VALUE *this_, ARRAY *arguments)
#define RETURN_OUT(v) v.return_ = true; return v;
#define GetHandle(v) ((HANDLE)((SINT)((VALUE *)ValueGetProperty(v, "handle"))->number))

enum {
	VALUE_NUMBER = 0,
	VALUE_STRING,
	VALUE_ARRAY,
	VALUE_FUNCTION,
	VALUE_COMPILED_FUNCTION,
	VALUE_BREAK,
	VALUE_SCOPE
};

typedef struct {
	unsigned char type;
	char *name;
	int stack_id;
	bool return_;

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

	ARRAY properties;
} VALUE;

void SetupLibraries();

VALUE Eval(TREE *tree, int stack_id);
VALUE ValueNumber(double number);
VALUE ValueString(char *string_);
VALUE ValueRawString(char *string_);
VALUE ValueArray();
VALUE ValueFunction(TREE *function);
VALUE ValueScope();
VALUE ValueCopy(VALUE *value);
VALUE ValueCompiledFunction(void *function);
void PrintValue(VALUE *value, bool quotes);
void ValueFree(VALUE *value);
void ValueFreeEx(VALUE *value);
VALUE *ValueGetProperty(VALUE *value, char *name);
VALUE *ValueSetProperty(VALUE *value, char *name, VALUE *property);

bool StackContains(char *name, int stack_id);
VALUE *StackPush(char *name, VALUE *value, int stack_id);
VALUE *StackGet(char *name, int stack_id);
VALUE *StackGetWithProperties(TREE *var, int stack_id);
void StackSet(TREE *var, VALUE *value, int stack_id);