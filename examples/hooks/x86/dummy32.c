// Make sure optimization is disabled so that the functions are actually called because the compiler will notice that the return values are constant
#include <stdio.h>
#include <Windows.h>

int __cdecl add(int a, int b) {
	return a + b;
}

long long __stdcall subtract(long long a, long long b) {
	return a - b;
}

double __fastcall divide(float a, double b) {
	return a / b;
}

// Small workaround for __thiscall in C
float __fastcall multiply(int a, void *_, short b) {
	return (float)(a * b);
}

int main() {
	static char buffer[0xFF] = { 0x12, 0x34, 0x56, 0x78, 0x91 };
	int offset = (int)buffer + 5;

	*(int *)offset = (int)add;
	offset += 4;

	*(int *)offset = (int)subtract;
	offset += 4;

	*(int *)offset = (int)divide;
	offset += 4;

	*(int *)offset = (int)multiply;

	for (;;) {
		printf("Press any key to call the functions");
		getchar();

		printf("\tadd(1,2) = %d\n", add(1, 2));
		printf("\tsubtract(3,4) = %lld\n", subtract(3, 4));
		printf("\tdivide(5,6) = %f\n", divide(5, 6));
		printf("\tmultiply(7,8) = %f\n\n", multiply(7, 0, 8));
	}
}