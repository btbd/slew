#pragma once

#define _CRT_SECURE_NO_WARNINGS
#define _WINSOCK_DEPRECATED_NO_WARNINGS

#include <io.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <regex>
#include <winsock2.h>
#include <ws2tcpip.h>
#include <Windows.h>

#pragma comment(lib, "Winmm.lib")
#pragma comment (lib, "Ws2_32.lib")

#ifdef _WIN64
#define SINT unsigned long long
#else
#define SINT unsigned int
#endif

#include "array.h"
#include "syntax.h"
#include "tree.h"
#include "eval.h"
#include "memory.h"

using namespace std;