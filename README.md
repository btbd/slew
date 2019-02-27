# Slew

An interpreted language designed for WinAPI.

## Building

### Prerequisites
- 64bit Windows OS (for the proper WinAPI libraries)
- 64bit GCC compiler
- [Go](https://golang.org/dl/) for Windows
- [make](http://gnuwin32.sourceforge.net/packages/make.htm) for Windows

If you have make, just do `make`. Or run the build command in the `Makefile`.

To run a program, do `slew.exe <file>`.

To create an executable, do `slew.exe -o <output> <file>`

## Examples

### Hello, world!
```js
console.Println("Hello, world!"); // output: Hello, world!
```

### Values

```js
console.Println("string");
console.Println(5, -23.58, true, false);
console.Println({
    prop1: "value",
    "another property": 27
});
console.Println([1, 2, "hello", 6, ["a", "b"]]);

/* output:

string
5 -23.58 1 0
{prop1:value,another property:27}
[1,2,hello,6,[a,b]]

*/
```

### Variables
```js
a := 5; // use := to declare a variable in a new scope
b := 6;

a = a + b; // use = to set a previously declared variable

console.Println(a);

b = 10;
a = a + b;
console.Println(a, b);

/* output:

11
21 10

*/
```

### If/else statements
```js
if (5 % 2 == 0) {
    console.Println("5 is divisible by two");
} else if (7 % 2 == 0) {
    console.Println("7 is divisible by two");
} else {
    console.Println("neither 5 nor 7 is divisible by two");
}

// output: neither 5 nor 7 is divisible by two
```

### Ternary operator
```js
console.Println(2 + 3 == 5 && 6 - 1 == 5 ? "yes" : "no"); // output: yes
```

### Loops
```js
for (i := 0; i < 10; ++i) {
    if (i > 0) {
        console.Print(", ")
    }
    console.Print(i)
}

console.Println();

i := 0;
while (i++ < 10) {
    console.Print(i + " ");
}

/* output:

0, 1, 2, 3, 4, 5, 6, 7, 8, 9
1 2 3 4 5 6 7 8 9 10

*/
```

### Functions

```js
func fib(n) {
    return n < 2 ? n : fib(--n) + fib(--n);
}

console.Println(fib(25));

args := func() {
    for (i := 0; i < len(arguments); ++i) {
        console.Print(arguments[i] + " ");
    }
    console.Println();
}

args(1, 2, "three", 4);

obj := {
    method: func() {
        console.Println(this.something);
    },
    something: "this is something",
};

obj.method();

/* output:

75025
1 2 three 4
this is something

*/
```

### Strings

```js
a := "thing";
a[0] = "somet";
console.Println(a);

a += '\x31\u03BC\n"something":\t世界\n';

for (i := 0; i < len(a); ++i) {
    console.Print(a[i]);
}

/* output:

something
something1μ
"something":    世界

*/
```

### Objects

```js
a := {
    "uno one": 1,
    "dos": {
        test: 2, // trailing comma is allowed
    }
};

if (a.dos.test == a["dos"]["test"]) {
    console.Println(a["uno one"]);
}

// output: 1
```

Find more examples [here](https://github.com/btbd/slew/tree/master/examples).

## Documentation

### Globals

`true` - Alternative to the number `1`.

`false` - Alternative to the number `0`.

`len([array|object|string])`
- Returns the length of the given variable.
- If the argument is of type `array`, the return value is the number of variables in the array.
- If the argument is of type `object`, the return value is the number of key/value pairs in the object.
- If the argument is of type `string`, the return value is the number of characters in the string excluding the null-terminator.

`type(var)`
- Returns the type of `var` as a string.

`copy(var)`
- Returns a copy of `var`.

### Libraries

- [array](#array)
- [console](#console)
- [date](#date)
- [http](#http)
- [input](#input)
- [json](#json)
- [math](#math)
- [number](#number)
- [object](#object)
- [process](#process)
- [regex](#regex)
- [string](#string)
- [thread](#thread)

### array

`.Each(arr, callback)`

- For each variable in `arr`, the callback is called with `(variable, index)`. No return value.

`.Find(arr, callback)`

- For each variable in `arr`, the callback is called with `(variable, index)` until `callback` returns a non-false value. Returns the found variable.

`.Insert(arr, index, var [, var...])`

- Inserts variable(s) at the specified `index` in `arr`. Returns the new length of `arr.`

`.Pop(arr)`

- Pops off the last variable in `arr` and returns it. 

`.Push(arr, var [, var...])`

- Pushes variable(s) to the back of `arr`. Returns the new length of `arr`.

`.Remove(arr, index)`

- Removes the variable at `index` in `arr` and returns it.

`.Sort(arr, comparison)`

- Sorts `arr` by using `comparison`. The comparison function should expect to be called with `(a,b)` where `a` and `b` are variables needed to be sorted. The comparison function should return a positive number if `a` should shift right or a negative number if `a` should shift left.

- No return value.

### console

`.Print(var [, var...])`

- Prints the variable(s) to stdout. No return value.

`.Println(var [, var...])`

- Prints the variables(s) to stdout and prints a newline. No return value.

`.ReadLine()`

- Reads from stdin until a newline character. Returns the read buffer as a string.

`.Clear()`

- Clears the console. No return value.

### date

`.Now()`

- Returns the current time in milliseconds.

`.Time()`

- Returns an object containing the following properties about the system time.
    - `.Milliseconds` - The millisecond (0-999).
    - `.Seconds` - The second (0-59).
    - `.Minute` - The minute (0-59).
    - `.Hour` - The hour (0-23).
    - `.Day` - The day of the month (1-31).
    - `.DayOfWeek` - The day of the week (0-6) where 0 is Sunday and 6 is Saturday.
    - `.Month` - The month (1-12) where 1 is January and 12 is December.
    - `.Year` - The year (1601-30827).

### http

`.Get(url)`

- Sends a GET request to `url`. Returns the body of the HTTP response.

`.Request(req)`

- Does the specified HTTP request. Returns an object containing the `Body` and `Status` of the HTTP response.
- `req` must be an object with the following properties:
    - `Body`
        - The body of the request.
    - `Headers`
        - An object of headers in the form of `Header:Value` .
    - `Method`
        - The method of request.
    - `URL`
        - The URL that receives the request.
- Example:

```js
resp := http.Request({
    URL: "https://somehost.com",
    Body: "some text for the body",
    Method: "POST",
    Headers: {
        Authorization: "Basic 123454321"
    }
});

console.Println("Status: " + resp.Status + "\nBody: " + resp.Body);
```

### input

`.IsKeyDown(key)`
- Returns a boolean value representing whether `key` is pressed.

`.KeyDown(key)`
- Sends a global keydown event for `key`. No return value.

`.KeyUp(key)`
- Sends a global keyup event for `key`. No return value.

`.OnKeyDown(callback)`
- On every global keydown event, `callback` is called with `(key)`. No return value.

`.OnKeyUp(callback)`
- On every global keyup event, `callback` is called with `(key)`. No return value.

`.SendKeys(keys)`
- Sends a keydown and keyup event for each key specified in `keys`.
- `keys` is a string representing the keys (not keycodes) to be pressed.
- Example:
    - `input.SendKeys("hello world");`

### json

`.Stringify(var)`
- Returns the string JSON format of `var`.

`.Parse(string)`
- Returns the variable format of the string JSON input.

### math

`.DEG_RAD` - Number of degrees per radian.

`.E` - Euler's constant and the base of natural logarithms, approximately 2.718.

`.LN10` - Natural logarithm of 10, approximately 2.303.

`.LN2` - Natural logarithm of 2, approximately 0.693.

`.LOG10E` - Base 10 logarithm of E, approximately 0.434.

`.LOG2E` - Base 2 logarithm of E, approximately 1.443.

`.PHI` - The golden ratio, approximately 1.618.

`.PI` - Ratio of the circumference of a circle to its diameter, approximately 3.14159.

`.RAD_DEG` - Number of radians per degree.

`.SQRT1_2` - Square root of 1/2, approximately 0.707.

`.SQRT2` - Square root of 2, approximately 1.414.

`.Abs(x)`
- Returns the absolute value of `x`.

`.Acos(x)`
- Returns the arccosine of `x`.

`.Acosh(x)`
- Returns the hyperbolic arccosine of `x`.

`.Asin(x)`
- Returns the arcsine of `x`.

`.Atan(x)`
- Returns the arctangent of `x`.

`.Atan2(y, x)`
- Returns the arctangent of the quotient of `y` and `x`.

`.Cbrt(x)`
- Returns the cube root of `x`.

`.Ceil(x)`
- Returns the smallest integer greater than or equal to `x`.

`.Cos(x)`
- Returns the cosine of `x`.

`.Cosh(x)`
- Returns the hyperbolic cosine of `x`.

`.Exp(x)`
- Returns E (Euler's constant) to the power of `x`.

`.Expm1(x)`
- Returns subtracting 1 from `math.Exp(x)`.

`.Floor(x)`
- Returns the largest integer less than or equal to `x`.

`.Hypot([x, [, y...]])`
- Returns the square root of the sum of squares of the arguments.

`.Log(x)`
- Returns the natural logarithm of `x`.

`.Log10(x)`
- Returns the base 10 logarithm of `x`.

`.Log1p(x)`
- Returns the natural logarithm of `1 + x`.

`.Log2(x)`
- Returns the base 2 logarithm of `x`.

`.Max([x, [, y...])`
- Returns the largest of zero or more arguments.

`.Min([x, [, y...])`
- Returns the smallest of zero or more arguments.

`.Pow(x, y)`
- Returns `x` to the power of `y`.

`.Random()`
- Returns a pseudo-random number between 0 and 1.

`.Round(x)`
- Returns the value of `x` rounded to the nearest integer.

`.Sign(x)`
- Returns `-1`, `0`, or `1` depending on the sign of `x`.

`.Sin(x)`
- Returns the sine of `x`.

`.Sinh(x)`
- Returns the hyperbolic sine of `x`.

`.Sqrt(x)`
- Returns the positive square root of `x`.

`.Tan(x)`
- Returns the tangent of `x`.

`.Tanh(x)`
- Returns the hyperbolic tangent of `x`.

`.Trunc(x)`
- Returns the integer part of `x`, removing any fractional digits.

### number

`.ToString(number, [base])`
- Returns the base 10, or specified base, string representation of `number`.

`.FromString(string [, base])`
- Returns the base 10, or specified base, number converted from its string representation.

`.ToInt16Bytes(number)`
- Returns an array of byte representing the 16bit integer in little-endian format.

`.ToUint16Bytes(number)`
- Returns an array of byte representing the 16bit unsigned integer in little-endian format.

`.ToInt32Bytes(number)`, `.ToIntBytes(number)`
- Returns an array of byte representing the 32bit integer in little-endian format.

`.ToUint32Bytes(number)`, `.ToUintBytes(number)` 
- Returns an array of byte representing the 32bit unsigned integer in little-endian format.

`.ToInt64Bytes(number)`
- Returns an array of byte representing the 64bit integer in little-endian format.

`.ToUint64Bytes(number)`
- Returns an array of byte representing the 64bit unsigned integer in little-endian format.

`.ToFloat32Bytes(number)`, `.ToFloatBytes(number)`
- Returns an array of byte representing the 32bit float.

`.ToFloat64Bytes(number)`, `.ToDoubleBytes(number)`
- Returns an array of byte representing the 64bit float.

`.FromInt16Bytes(bytes)`
- Converts an array of 2 bytes to a 16bit integer and returns the result.

`.FromUint16Bytes(bytes)`
- Converts an array of 2 bytes to a 16bit unsigned integer and returns the result.

`.FromInt32Bytes(bytes)`, `.FromIntBytes(bytes)`
- Converts an array of 4 bytes to a 32bit integer and returns the result.

`.FromUint32Bytes(bytes)`, `.FromUintBytes(bytes)`
- Converts an array of 4 bytes to a 32bit unsigned integer and returns the result.

`.FromInt64Bytes(bytes)`
- Converts an array of 8 bytes to a 64bit integer and returns the result.

`.FromUint64Bytes(bytes)`
- Converts an array of 8 bytes to a 64bit unsigned integer and returns the result.

`.FromFloat32Bytes(bytes)`, `.FromFloatBytes(bytes)`
- Converts an array of 4 bytes to a 32bit float and returns the result.

`.FromFloat64Bytes(bytes)`, `.FromDoubleBytes(bytes)`
- Converts an array of 8 bytes to a 64bit float and returns the result.

### object

`.Keys(object)`
- Returns an array of all the keys in `object`.

### process

`.Open([exe|pid])`

- Returns a [process object](#process-object) of the specified process found either by name or process identifier.

`.List()`

- Returns an array of all system processes, each with the following properties:
    - `.Id`
        - The identifier of the process.e
    - `.Name`
        - The name of the process. 
    - `.ParentId`
        - The identifier of the process that created this process (its parent process).
    - `.PriClassBase`
        - The base priority of any threads created by this process.
    - `.Threads`
        - The number of execution threads at the time `process.List()` was called.

`.Current`

- The [process object](#process-object) for the current process.

#### Process Object

- `.Handle` - The Windows handle to the process.
- `.Id` - The identifier of the process.
- `.Name` - The name of the process.
- `.ParentId` - The identifier of the process that created this process (its parent process).
- `.PriClassBase` - The base priority of any threads created by this process.
- `.Wow64` - Boolean value that indicates whether the process is running under WOW64 (32bit process).
- `.Alloc(size [, allocType [, protect]])`
    - Allocates a region of memory of `size` in the process and sets it to zero.
    - `allocType` is the type of memory allocation (default `process.MEM_COMMIT | process.MEM_RESERVE`). It must contain one of the following values:
        - `process.MEM_COMMIT`
        - `process.MEM_RESERVE`
        - Optional:
        - `process.MEM_LARGE_PAGES`
        - `process.MEM_PHYSICAL`
        - `process.MEM_TOP_DOWN`
    - `protect` is the memory protection for the region of pages to be allocated (default `process.PAGE_EXECUTE_READWRITE`). It can be any of the [memory protection constants](#memory-protection-constants-full-doc).
    - Returns the base address of the allocated region.
- `.Call(address [, flags [, args...]])`
    - Calls a compiled function in the process at `address`.
    - `flags` specifies the type of function to call.
        - Calling conventions for a WOW64 or 32bit process (default `process.FUNC_CDECL`)
            - `process.FUNC_CDECL`
            - `process.FUNC_STDCALL`
            - `process.FUNC_FASTCALL`
            - `process.FUNC_THISCALL`
        - Note: for a 64bit process, `__fastcall` is the only calling convention so no calling convention flag needs to be specified.
        - Return types (default `process.FUNC_RET_INT32`):
            - `process.FUNC_RET_INT32`, `process.FUNC_RET_INT`
            - `process.FUNC_RET_INT64`
            - `process.FUNC_RET_FLOAT32`, `process.FUNC_RET_FLOAT`
            - `process.FUNC_RET_FLOAT64`
            - `process.FUNC_RET_NONE`
            - Optional:
            - `process.FUNC_RET_RAW`
    - `args` are the arguments to pass to the function. They must be objects containing a key/value pair for `Type` and `Value`.
        - `Type` specifies the type of argument and can be any one of the following values:
            - `process.ARG_INT8`
            - `process.ARG_INT16`
            - `process.ARG_INT32`, `process.ARG_INT`
            - `process.ARG_INT64`
            - `process.ARG_FLOAT32`, `process.ARG_FLOAT`
            - `process.ARG_FLOAT64`
            - Optional:
            - `process.ARG_RAW`
        - `Value` can be any number value corresponding to `Type` unless `process.ARG_RAW` is specified. If `process.ARG_RAW` is specified, `Value` must be an array of bytes corresponding to the argument's type.
    - Returns the specified return value once the virtual thread is complete. If `process.FUNC_RET_NONE` is given, then the function returns immediately.
    - Example:

```js
p := process.Open("test.exe"); // test.exe is a 32bit process

ret := p.Call(
    0xBEEF,
    process.FUNC_STDCALL | process.FUNC_RET_FLOAT32,
    { Type: process.ARG_FLOAT64, Value: 12.21 },
    { Type: process.ARG_INT32 | process.ARG_RAW, Value: [ 32, 0, 0, 0 ]}
);

console.Println(ret);
```

- `.Close()`
    - Closes the Windows handle associated with the process. No return value.
- `.Exit([exitCode])`
    - Terminates the process with an optional exit code (default `0`). No return value.
- `.FindPattern(pattern [, mask])`
    - Finds a byte pattern in the virtual memory of the process and returns the base address of the first match.
    - If `mask` is not specified, `pattern` must be one of the following:
        - A string with a two character hex representation for each known byte and `??` for each unknown byte. Spaces are automatically ignored. Example:
            - `p.FindPattern("12 43 5A 68 1F ?? ?? 59 23 ?? 0F");`
        - An array of byte with no unknown bytes. Example:
            - `p.FindPattern([0x59, 0x5A, 0x34, 0x5F, 0x9B]);`
    - If `mask` is specified, `pattern` must be an array of byte or a string that will be treated as an array of byte. `mask` must be a string with an `x` for a known byte and any other ASCII character for an unknown byte. Examples:
        - `p.FindPattern("\x12\x43\x5A\x68\x1F", "xx??x");`
        - `p.FindPattern([0x12, 0x43, 0x5A, 0x68, 0x1F], "xx??x");`
- `.Free(address, [, size [, freeType]])`
    - Frees a region of memory in the process.
    - If `freeType` is `process.MEM_RELEASE`, then `size` must be zero (default `0`).
    - `freeType` can be one of the following values (default `process.MEM_RELEASE`):
        - `process.MEM_COALESCE_PLACEHOLDERS`
        - `process.MEM_DECOMMIT`
        - `process.MEM_PRESERVE_PLACEHOLDER`
        - `process.MEM_RELEASE`
    - Returns a non-zero value on success.
- `.GetProcAddress(module, proc)`
    - Returns the address of the specified procedure in `module` in the process.
    - This is the C equivalent of `GetProcAddress(GetModuleHandleA(module), proc);` in the address space of the process.
    - Example:
        - `p.GetProcAddress("kernel32.dll", "LoadLibraryA");`
- `.LoadLibrary(library)`
    - Loads the specified module into the address space of the process. No return value.
    - This is the C equivalent of `LoadLibraryA(library);` in the address space of the process.
- `.Modules([name])`
    - Returns an array of [module object](#module-object) for each module or the specified module in the process.
- `.Protect(address, size, protect)`
    - Changes the protection on a region of committed pages in the virtual address space of the process from `address` to `address+size`. Returns the previous protection of the first page in the specified region of pages.
    - `protect` can be any of the [memory protection constants](#memory-protection-constants-full-doc).
- `.Read(address, size)`
    - Returns an array of byte of `size` at the specified address.
- `.ReadFloat32(address)`, `.ReadFloat(address)`
    - Returns the 32bit float representation of the 4 bytes at `address`.
- `.ReadFloat64(address)`, `.ReadDouble(address)`
    - Returns the 64bit float representation of the 8 bytes at `address`.
- `.ReadInt16(address)`
    - Returns the 16bit float representation of the 2 bytes at `address`.
-  `.ReadInt32(address)`, `.ReadInt(address)`
    - Returns the 32bit integer representation of the 4 bytes at `address`.
- `.ReadInt64(address)`
    - Returns the 64bit integer representation of the 8 bytes at `address`.
- `.ReadPointer([offset...])`
    - Follows the specified offset list and returns the final pointer.
- `.ReadString(address)`, `.ReadString8(address)`
    - Returns the ASCII string at `address`. The length is determined by finding a null-terminator.
- `.ReadString16(address)`
    - Returns the Unicode string at `address`. The length is determined by finding a null-terminator.
- `.ReadUint16(address)`
    - Returns the 16bit unsigned integer representation of the 2 bytes at `address`.
- `.ReadUint32(address)`, `.ReadUint(address)`
    - Returns the 32bit unsigned integer representation of the 4 bytes at `address`.
- `.ReadUint64(address)`
    - Returns the 64bit unsigned integer representation of the 8 bytes at `address`.
- `.Resume()`
    - Resumes execution of the process.
- `.Suspend()`
    - Suspends execution of the process.
- `.Threads()`
    - Returns an array of [thread object](#thread-object) for each execution thread in the process.
- `.Write(address, bytes)`
    - Writes an array of byte to `address`.
    - Returns a non-zero value on success.
- `.WriteFloat32(address, float32)`, `.WriteFloat(address, float)`
    - Writes a 32bit float to `address`.
    - Returns a non-zero value on success.
- `.WriteFloat64(address, float64)`, `.WriteDouble(address, double)`
    - Writes a 64bit float to `address`.
    - Returns a non-zero value on success.
- `.WriteInt16(address, int16)`
    - Writes a 16bit integer to `address`.
    - Returns a non-zero value on success.
- `.WriteInt32(address, int32)`, `.WriteInt(address, int)`
    - Writes a 32bit integer to `address`.
    - Returns a non-zero value on success.
- `.WriteInt64(address, int64)`
    - Writes a 64bit integer to `address`.
    - Returns a non-zero value on success.
- `.WriteString(address, string)`, `.WriteString8(address, string)`
    - Writes an ASCII string to `address` (including null-terminator).
    - Returns a non-zero value on success.
- `.WriteString16(address, string)`
    - Writes a Unicode string to `address` (including null-terminator).
    - Returns a non-zero value on success.
- `.WriteUint16(address, uint16)`
    - Writes a 16bit unsigned integer to `address`.
    - Returns a non-zero value on success.
- `.WriteUint32(address, uint32)`, `.WriteUint(address, uint)`
    - Writes a 32bit unsigned integer to `address`.
    - Returns a non-zero value on success.
- `.WriteUint64(address, uint64)`
    - Writes a 64bit unsigned integer to `address`.
    - Returns a non-zero value on success.

#### Module Object

- `.Base` - The base address of the module in the address space of the process.
- `.Name` - The name of the module.
- `.Size` - The size of the module in bytes.

#### Thread Object

- `.CreationTime` - The creation time of the thread in 100-nanosecond intervals since January 1, 1601 (UTC).
- `.Id` - The identifier of the thread.
- `.Owner` - The identifier of the process that created the thread.
- `.Priority` - The kernel base priority level assigned to the thread.
- `.Stack` - The address of the bottom of the thread's stack.
- `.Resume()`
    - Resumes execution of the thread. Returns a non-zero value on success.
- `.Suspend()`
    - Suspends execution of the thread. Returns a non-zero value on success.

#### Memory Protection Constants ([full doc](https://docs.microsoft.com/en-us/windows/desktop/memory/memory-protection-constants))

- `process.PAGE_EXECUTE_READ` - Enables execute or read-only access to the committed region of pages.
- `process.PAGE_EXECUTE_READWRITE` - Enables execute, read-only, or read/write access to the committed region of pages.
- `process.PAGE_EXECUTE_WRITECOPY` - Enables execute, read-only, or copy-on-write access to a mapped view of a file mapping object.
- `process.PAGE_EXECUTE` - Enables execute access to the committed region of pages.
- `process.PAGE_GUARD` - Pages in the region become guard pages.
- `process.PAGE_NOACCESS` - Disables all access to the committed region of pages.
- `process.PAGE_NOCACHE` - Sets all pages to be non-cachable.
- `process.PAGE_READONLY` - Enables read-only access to the committed region of pages.
- `process.PAGE_READWRITE` - Enables read-only or read/write access to the committed region of pages.
- `process.PAGE_TARGETS_INVALID` - Sets all locations in th epages as invalid targets for CFG.
- `process.PAGE_TARGETS_NO_UPDATE` - Pages in the region will not have their CFG information updated while the protection changes.
- `process.PAGE_WRITECOMBINE` - Sets all pages to be write-combined.
- `process.PAGE_WRITECOPY` - Allows views to be mapped for read-only, copy-on-write, or execute access.

### regex

`.Find(string, regex)`
- Returns all matches of `regex` in `string`.

`.FindIndex(string, regex)`
- Returns an array of an array of two indexes representing each match of `regex`.
- The first index is the start of the match, and the second index is the end of the match.

`.Match(string, regex)`
- Returns a non-zero value if `string` contains any match of `regex`.

`.Replace(string, regex, replacement [, count])`
- Returns `string` with all (or specified `count`) matches of `regex` replaced with `replacement`.

`.Split(string, regex [,count])`
- Slices `string` into substrings separated by all (or specified by `count`) matches of `regex` and returns an array of the substrings between the matches.

### string

`.CharCodeAt(string, index)`
- Returns the character code at `index` in `string`.

`.Contains(string, substring)`
- Returns a non-zero value if `string` contains `substring`.

`.FromCharCode(code [, code...])`
- Returns a string created from the specified character codes.

`.FromNumber(number, [base])`
- Returns the base 10, or specified base, string representation of `number`.

`.IndexOf(string, substring)`
- Returns the first index of `substring` in `string`.

`.LastIndexOf(string, substring)`
- Returns the last index of `substring` in `string`.

`.Replace(string, substring, replacement [, count])`
- Replaces all (or `count`) occurrences of `substring` in `string` with `replacement` and returns the result.

`.Slice(string, start [, end])`
- Returns a substring of `string` from `start` to `end`.
- If `end` is not specified, `end` is the length of the string.
- If `end` is negative, `end` is `length + end`.

`.Split(string, delimiter)`
- Returns an array of substrings from `string` that were between `delimiter`.

`.ToLower(string)`
- Returns `string` to lowercase 

`.ToNumber`
- Returns `string` to lowercase 

`.ToUpper`
- Returns `string` to uppercase 

`.Trim(string , [cutset])`
- Returns `string` with whitespace removed from both sides. 
- If `cutset` is specified, all characters in the string `cutset` will be removed from both sides of `string`.

### thread

`.Create(callback)`
- Creates a new thread that executes `callback`.

`.Sleep(ms)`
- Halts execution of the calling thread for `ms` milliseconds.