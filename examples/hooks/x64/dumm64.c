// Make sure optimization is disabled so that the functions are actually called because the compiler will notice that the return values are constant
#include <stdio.h>
#include <Windows.h>

int add(int a, int b) {
	return a + b;
}

long long subtract(long long a, long long b) {
	return a - b;
}

double divide(float a, double b) {
	return a / b;
}

float multiply(int a, short b) {
	return (float)(a * b);
}

int main() {
	static char buffer[0xFF] = { 0x12, 0x34, 0x56, 0x78, 0x91 };
	ULONG64 offset = (ULONG64)buffer + 5;

	*(ULONG64 *)offset = (ULONG64)add;
	offset += 8;

	*(ULONG64 *)offset = (ULONG64)subtract;
	offset += 8;

	*(ULONG64 *)offset = (ULONG64)divide;
	offset += 8;

	*(ULONG64 *)offset = (ULONG64)multiply;

	for (;;) {
		printf("Press any key to call the functions");
		getchar();

		printf("\tadd(1,2) = %d\n", add(1, 2));
		printf("\tsubtract(3,4) = %lld\n", subtract(3, 4));
		printf("\tdivide(5,6) = %f\n", divide(5, 6));
		printf("\tmultiply(7,8) = %f\n\n", multiply(7, 8));
	}
}