#pragma once

typedef struct {
	void *buffer;
	unsigned char element_size;
	unsigned int length, allocated;
} ARRAY;

ARRAY ArrayNew(unsigned char element_size);
void *ArrayGet(ARRAY *array, DWORD index);
void *ArraySet(ARRAY *array, DWORD index, void *element);
void *ArrayPush(ARRAY *array, void *element);
void ArrayPop(ARRAY *array, void *out);
void ArrayMerge(ARRAY *dest, ARRAY *array);
void ArrayFree(ARRAY *array);