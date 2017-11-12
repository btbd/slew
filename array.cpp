#include "main.h"

ARRAY ArrayNew(unsigned char element_size) {
	ARRAY array;

	array.element_size = element_size;
	array.allocated = 0xFF;
	array.buffer = malloc(element_size * 0xFF);
	array.length = 0;

	return array;
}

void *ArrayGet(ARRAY *array, DWORD index) {
	return (void *)((SINT)array->buffer + (index * array->element_size));
}

void *ArraySet(ARRAY *array, DWORD index, void *element) {
	return memcpy((void *)((SINT)array->buffer + (index * array->element_size)), element, array->element_size);
}

void *ArrayPush(ARRAY *array, void *element) {
	if (array->length >= array->allocated) {
		array->allocated *= 2;
		array->buffer = realloc(array->buffer, array->allocated * array->element_size);
	}

	return memcpy((void *)((SINT)array->buffer + (array->length++ * array->element_size)), element, array->element_size);
}

void ArrayPop(ARRAY *array, void *out) {
	if (out) {
		memcpy(out, ArrayGet(array, --array->length), array->element_size);
	} else {
		--array->length;
	}
}

void ArrayMerge(ARRAY *dest, ARRAY *array) {
	dest->allocated += array->allocated;
	dest->buffer = realloc(dest->buffer, dest->allocated * array->element_size);

	memcpy((void *)((SINT)dest->buffer + (dest->length * dest->element_size)), array->buffer, array->length * dest->element_size);

	dest->length += array->length;
}

void ArrayFree(ARRAY *array) {
	if (array->buffer) {
		free(array->buffer);
		array->buffer = 0;
	}
}