#pragma once

typedef struct {
	ARRAY stack;
	int access_stack;
} THREAD;

typedef struct {
	TREE *func;
	THREAD thread;
	ARRAY *arguments;
} THREAD_ARGUMENTS;

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
	VALUE_SCOPE,
	VALUE_STACK
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

void SetupLibraries(THREAD *thread);

VALUE Eval(THREAD *thread, TREE *tree, int stack_id);
VALUE ValueNumber(double number);
VALUE ValueString(char *string_);
VALUE ValueRawString(char *string_);
VALUE ValueArray();
VALUE ValueFunction(THREAD *thread, TREE *function);
VALUE ValueScope(THREAD *thread);
VALUE ValueCopy(VALUE *value);
VALUE ValueCompiledFunction(void *function);
VALUE ValueStack();
void PrintValue(VALUE *value, bool quotes);
void ValueFree(VALUE *value);
void ValueFreeEx(VALUE *value);
VALUE *ValueGetProperty(VALUE *value, char *name);
VALUE *ValueSetProperty(VALUE *value, char *name, VALUE *property);

bool StackContains(THREAD *thread, char *name, int stack_id);
VALUE *StackPush(THREAD *thread, char *name, VALUE *value, int stack_id);
VALUE *StackGet(THREAD *thread, char *name, int stack_id);
VALUE *StackGetWithProperties(THREAD *thread, TREE *var, int stack_id);
void StackSet(THREAD *thread, TREE *var, VALUE *value, int stack_id);