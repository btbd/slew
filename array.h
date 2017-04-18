#pragma once

#define ARRAY_GET(a, i) a->buffer + (i * a->element_size)
#define ARRAY_SET(a, i, e) memcpy(ARRAY_GET(a, i), e, a->element_size)

typedef struct {
	char *buffer;

	unsigned char element_size;
	unsigned int length, _allocated;
} ARRAY;

ARRAY *Array_New(unsigned char element_size);
void *Array_Push(ARRAY *array, void *element);
void Array_Pop(ARRAY *array, void *element_out);
void *Array_Set(ARRAY *array, unsigned int index, void *element);
void *Array_Get(ARRAY *array, unsigned int index);
void Array_Free(ARRAY *array);