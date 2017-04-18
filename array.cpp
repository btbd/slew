#include "main.h"

// Creates a new array with size of element
ARRAY *Array_New(unsigned char element_size) {
	ARRAY *array = (ARRAY *)malloc(sizeof(ARRAY));

	array->element_size = element_size;
	array->length = 0;
	array->_allocated = 0xFF;
	array->buffer = (char *)malloc(array->_allocated * element_size);

	return array;
}

// Pushes an element to the end of an array and returns the element
void *Array_Push(ARRAY *array, void *element) {
	if (array->length >= array->_allocated) {
		array->_allocated *= 2;
		array->buffer = (char *)realloc(array->buffer, array->_allocated * array->element_size);
	}

	return ARRAY_SET(array, array->length++, element);
}

// Removes and gets the last element of an array
void Array_Pop(ARRAY *array, void *element_out) {
	if (array->length > 0) {
		if (element_out) {
			memcpy(element_out, ARRAY_GET(array, --array->length), array->element_size);
		} else {
			--array->length;
		}
	}
}

// Sets an element at a specific index in an array
void *Array_Set(ARRAY *array, unsigned int index, void *element) {
	if (index >= array->_allocated) {
		array->_allocated = index + 1;
		array->buffer = (char *)realloc(array->buffer, array->_allocated * array->element_size);
	}

	if (index >= array->length) {
		array->length = index + 1;
	}

	return ARRAY_SET(array, index, element);
}

// Returns an element at a specific index in an array
void *Array_Get(ARRAY *array, unsigned int index) {
	return ARRAY_GET(array, index);
}

// Frees an array from memory
void Array_Free(ARRAY *array) {
	free(array->buffer);
	free(array);
}