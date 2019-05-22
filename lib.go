package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/JamesHovious/w32"
	"golang.org/x/sys/windows"
)

type MessageHook struct {
	ProcessHandle   w32.HANDLE
	ProcessId       uint32
	Address         uint64   // Address of hooked func
	ThreadIdAddress uint64   // Address for ID of calling thread
	Args            []uint32 // Flags for args
	ArgsAddress     uint64   // Address of stored args
	Ret             uint32   // Flag for ret type
	RetAddress      uint64   // Address to store ret
	Patch           []byte   // Original bytes
	Hook            uint64   // Address of hook
	Object          *Variable
	UserHandler     *Variable
}

var msgThreadId uint32
var msgHooks []MessageHook
var msgHooksMutex = &sync.RWMutex{}

func global_len(this *Variable, args []Variable) Variable {
	if len(args) > 0 {
		switch args[0].Type {
		case VAR_STRING:
			return MakeVariable(VAR_NUMBER, float64(len([]rune(args[0].Value.(string)))))
		case VAR_ARRAY:
			return MakeVariable(VAR_NUMBER, float64(len(*args[0].Value.(*[]Variable))))
		case VAR_OBJECT:
			return MakeVariable(VAR_NUMBER, float64(len(*args[0].Value.(*map[string]*Variable))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func global_type(this *Variable, args []Variable) Variable {
	s := "unknown"

	if len(args) > 0 {
		switch args[0].Type {
		case VAR_NUMBER:
			s = "number"
		case VAR_STRING:
			s = "string"
		case VAR_FUNCTION:
			fallthrough
		case VAR_NFUNCTION:
			s = "function"
		case VAR_ARRAY:
			s = "array"
		case VAR_OBJECT:
			s = "object"
		}
	}

	return MakeVariable(VAR_STRING, s)
}

func Copy(v Variable) Variable {
	switch v.Type {
	case VAR_NUMBER:
		fallthrough
	case VAR_STRING:
		fallthrough
	case VAR_FUNCTION:
		fallthrough
	case VAR_NFUNCTION:
		return v
	case VAR_ARRAY:
		var r []Variable
		for _, e := range *v.Value.(*[]Variable) {
			r = append(r, Copy(e))
		}
		return MakeVariable(VAR_ARRAY, &r)
	case VAR_OBJECT:
		m := map[string]*Variable{}
		for k, v := range *v.Value.(*map[string]*Variable) {
			cv := Copy(*v)
			m[k] = &cv
		}
		return MakeVariable(VAR_OBJECT, &m)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func global_copy(this *Variable, args []Variable) Variable {
	if len(args) > 0 {
		return Copy(args[0])
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                    object                   */
/***********************************************/
func object_Keys(this *Variable, args []Variable) Variable {
	var r []Variable
	if len(args) > 0 && args[0].Type == VAR_OBJECT {
		for k, _ := range *args[0].Value.(*map[string]*Variable) {
			r = append(r, MakeVariable(VAR_STRING, k))
		}
	}
	return MakeVariable(VAR_ARRAY, &r)
}

/***********************************************/
/*                      pop                    */
/***********************************************/
func array_Push(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_ARRAY {
		ptr := args[0].Value.(*[]Variable)
		arr := *ptr
		for i := 1; i < len(args); i++ {
			arr = append(arr, args[i])
		}
		*ptr = arr
		return MakeVariable(VAR_NUMBER, float64(len(arr)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Pop(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		ptr := args[0].Value.(*[]Variable)
		arr := *ptr
		if e := len(arr) - 1; e > -1 {
			r := arr[e]
			*ptr = arr[:e]
			return r
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Insert(this *Variable, args []Variable) Variable {
	if len(args) > 2 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_NUMBER {
		ptr := args[0].Value.(*[]Variable)
		arr := *ptr
		index := int(args[1].Value.(float64))
		if l := len(arr); index > -1 && index < l {
			for i := len(args) - 1; i > 1; i-- {
				arr = append(arr[:index], append([]Variable{args[i]}, arr[index:]...)...)
			}
		} else if index == l {
			for i := 2; i < len(args); i++ {
				arr = append(arr, args[i])
			}
		}

		*ptr = arr
		return MakeVariable(VAR_NUMBER, float64(len(arr)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Remove(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_NUMBER {
		ptr := args[0].Value.(*[]Variable)
		arr := *ptr
		index := int(args[1].Value.(float64))
		if index > -1 && index < len(arr) {
			r := arr[index]
			*ptr = append(arr[:index], arr[index+1:]...)
			return r
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Sort(this *Variable, args []Variable) Variable {
	if l := len(args); l > 1 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_FUNCTION {
		a := *args[0].Value.(*[]Variable)
		f := args[1]
		sort.SliceStable(a, func(i int, j int) bool {
			v := CallUserFunc(&f, nil, []Variable{a[i], a[j]})
			return !(v.Type == VAR_NUMBER && v.Value.(float64) > 0)
		})
	}
	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Find(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_FUNCTION {
		f := args[1]
		for i, v := range *args[0].Value.(*[]Variable) {
			if r := CallUserFunc(&f, nil, []Variable{v, MakeVariable(VAR_NUMBER, float64(i))}); r.Type != VAR_NUMBER || r.Value.(float64) != 0 {
				return v
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Each(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_FUNCTION {
		f := args[1]
		for i, v := range *args[0].Value.(*[]Variable) {
			if r := CallUserFunc(&f, nil, []Variable{v, MakeVariable(VAR_NUMBER, float64(i))}); r.Type != VAR_NUMBER || r.Value.(float64) != 0 {
				break
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                   console                   */
/***********************************************/
func console_Print(this *Variable, args []Variable) Variable {
	for i, e := range args {
		if i > 0 {
			fmt.Print(" " + ToString(e))
		} else {
			fmt.Print(ToString(e))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func console_Println(this *Variable, args []Variable) Variable {
	console_Print(this, args)
	fmt.Println()
	return MakeVariable(VAR_NUMBER, float64(0))
}

func console_ReadLine(this *Variable, args []Variable) Variable {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return MakeVariable(VAR_NUMBER, float64(0))
	}

	return MakeVariable(VAR_STRING, text)
}

func console_Clear(this *Variable, args []Variable) Variable {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                    number                   */
/***********************************************/
func number_ToString(this *Variable, args []Variable) Variable {
	if l := len(args); l > 0 && args[0].Type == VAR_NUMBER {
		if l > 1 && args[1].Type == VAR_NUMBER {
			return MakeVariable(VAR_STRING, strconv.FormatInt(int64(args[0].Value.(float64)), int(args[1].Value.(float64))))
		} else {
			return MakeVariable(VAR_STRING, strconv.FormatFloat(args[0].Value.(float64), 'f', -1, 64))
		}
	}

	return MakeVariable(VAR_STRING, "")
}

func number_ToInt16Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := int16(args[0].Value.(float64))
		bytes := *(*[2]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToUint16Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := uint16(args[0].Value.(float64))
		bytes := *(*[2]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToInt32Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := int32(args[0].Value.(float64))
		bytes := *(*[4]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToUint32Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := uint32(args[0].Value.(float64))
		bytes := *(*[4]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToInt64Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := int64(args[0].Value.(float64))
		bytes := *(*[8]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToUint64Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := uint64(args[0].Value.(float64))
		bytes := *(*[8]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToFloat32Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := float32(args[0].Value.(float64))
		bytes := *(*[4]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_ToFloat64Bytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		v := float64(args[0].Value.(float64))
		bytes := *(*[8]byte)(unsafe.Pointer(&v))

		for _, b := range bytes {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func number_FromInt16Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 1 {
			var b [2]byte
			for i := 0; i < 2; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*int16)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromUint16Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 1 {
			var b [2]byte
			for i := 0; i < 2; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*uint16)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromInt32Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 3 {
			var b [4]byte
			for i := 0; i < 4; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*int32)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromUint32Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 3 {
			var b [4]byte
			for i := 0; i < 4; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*uint32)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromInt64Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 7 {
			var b [8]byte
			for i := 0; i < 8; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*int64)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromUint64Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 7 {
			var b [8]byte
			for i := 0; i < 8; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*uint64)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromFloat32Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 3 {
			var b [4]byte
			for i := 0; i < 4; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*float32)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func number_FromFloat64Bytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		if a := *args[0].Value.(*[]Variable); len(a) > 7 {
			var b [8]byte
			for i := 0; i < 8; i++ {
				if a[i].Type != VAR_NUMBER {
					return MakeVariable(VAR_NUMBER, float64(0))
				}

				b[i] = byte(a[i].Value.(float64))
			}

			return MakeVariable(VAR_NUMBER, float64(*(*float64)(unsafe.Pointer(&b[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                    string                   */
/***********************************************/
func string_ToNumber(this *Variable, args []Variable) Variable {
	if l := len(args); l > 0 && args[0].Type == VAR_STRING {
		if l > 1 && args[1].Type == VAR_NUMBER {
			r, err := strconv.ParseInt(strings.TrimSpace(args[0].Value.(string)), int(args[1].Value.(float64)), 64)
			if err == nil {
				return MakeVariable(VAR_NUMBER, float64(r))
			}
		} else {
			r, err := strconv.ParseFloat(strings.TrimSpace(args[0].Value.(string)), 64)
			if err == nil {
				return MakeVariable(VAR_NUMBER, r)
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func string_FromBytes(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_ARRAY {
		var s []byte
		for _, b := range *args[0].Value.(*[]Variable) {
			if b.Type == VAR_NUMBER {
				s = append(s, byte(b.Value.(float64)))
			}
		}
		return MakeVariable(VAR_STRING, string(s))
	}
	return MakeVariable(VAR_STRING, "")
}

func string_FromCharCode(this *Variable, args []Variable) Variable {
	s := ""

	for _, v := range args {
		if v.Type == VAR_NUMBER {
			s += string(rune(int(v.Value.(float64))))
		} else {
			break
		}
	}

	return MakeVariable(VAR_STRING, s)
}

func string_CharCodeAt(this *Variable, args []Variable) Variable {
	c := 0

	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_NUMBER {
		s := []rune(args[0].Value.(string))
		i := int(args[1].Value.(float64))
		if i < len(s) {
			c = int(s[i])
		}
	}

	return MakeVariable(VAR_NUMBER, float64(c))
}

func string_Contains(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING && strings.Contains(args[0].Value.(string), args[1].Value.(string)) {
		return MakeVariable(VAR_NUMBER, float64(1))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func string_IndexOf(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		return MakeVariable(VAR_NUMBER, float64(strings.Index(args[0].Value.(string), args[1].Value.(string))))
	}

	return MakeVariable(VAR_NUMBER, float64(-1))
}

func string_LastIndexOf(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		return MakeVariable(VAR_NUMBER, float64(strings.LastIndex(args[0].Value.(string), args[1].Value.(string))))
	}

	return MakeVariable(VAR_NUMBER, float64(-1))
}

func string_Replace(this *Variable, args []Variable) Variable {
	if l := len(args); l > 2 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING && args[2].Type == VAR_STRING {
		n := -1
		if l > 3 && args[3].Type == VAR_NUMBER {
			n = int(args[3].Value.(float64))
		}

		return MakeVariable(VAR_STRING, strings.Replace(args[0].Value.(string), args[1].Value.(string), args[2].Value.(string), n))
	}

	return MakeVariable(VAR_STRING, "")
}

func string_Slice(this *Variable, args []Variable) Variable {
	if la := len(args); la > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_NUMBER {
		low := int(args[1].Value.(float64))
		high := 0
		if la > 2 && args[2].Type == VAR_NUMBER {
			high = int(args[2].Value.(float64))
		}

		s := []rune(args[0].Value.(string))
		l := len(s)

		if high < 1 {
			high = l + high
		}

		if low < high && low > -1 && high <= l {
			return MakeVariable(VAR_STRING, string(s[low:high]))
		}
	}

	return MakeVariable(VAR_STRING, "")
}

func string_Split(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		s := strings.Split(args[0].Value.(string), args[1].Value.(string))
		for _, e := range s {
			r = append(r, MakeVariable(VAR_STRING, e))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func string_ToBytes(this *Variable, args []Variable) Variable {
	var r []Variable

	if len(args) > 0 && args[0].Type == VAR_STRING {
		for _, b := range []byte(args[0].Value.(string)) {
			r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func string_ToUpper(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_STRING {
		return MakeVariable(VAR_STRING, strings.ToUpper(args[0].Value.(string)))
	}

	return MakeVariable(VAR_STRING, "")
}

func string_ToLower(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_STRING {
		return MakeVariable(VAR_STRING, strings.ToLower(args[0].Value.(string)))
	}

	return MakeVariable(VAR_STRING, "")
}

func string_Trim(this *Variable, args []Variable) Variable {
	if l := len(args); l > 0 && args[0].Type == VAR_STRING {
		if l > 1 && args[1].Type == VAR_STRING {
			return MakeVariable(VAR_STRING, strings.Trim(args[0].Value.(string), args[1].Value.(string)))
		} else {
			return MakeVariable(VAR_STRING, strings.TrimSpace(args[0].Value.(string)))
		}
	}

	return MakeVariable(VAR_STRING, "")
}

/***********************************************/
/*                    regex                    */
/***********************************************/
func regex_Match(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		if reg, err := regexp.Compile(args[1].Value.(string)); err == nil && reg.MatchString(args[0].Value.(string)) {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func regex_Find(this *Variable, args []Variable) Variable {
	var r []Variable

	if l := len(args); l > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		n := -1
		if l > 2 && args[2].Type == VAR_NUMBER {
			n = int(args[2].Value.(float64))
		}

		if reg, err := regexp.Compile(args[1].Value.(string)); err == nil {
			for _, s := range reg.FindAllString(args[0].Value.(string), n) {
				r = append(r, MakeVariable(VAR_STRING, s))
			}
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func regex_FindIndex(this *Variable, args []Variable) Variable {
	var r []Variable

	if l := len(args); l > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		n := -1
		if l > 2 && args[2].Type == VAR_NUMBER {
			n = int(args[2].Value.(float64))
		}

		if reg, err := regexp.Compile(args[1].Value.(string)); err == nil {
			for _, i := range reg.FindAllStringIndex(args[0].Value.(string), n) {
				r = append(r, MakeVariable(VAR_ARRAY, &[]Variable{MakeVariable(VAR_NUMBER, float64(i[0])), MakeVariable(VAR_NUMBER, float64(i[1]))}))
			}
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func regex_Replace(this *Variable, args []Variable) Variable {
	if l := len(args); l > 2 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING && args[2].Type == VAR_STRING {
		n := -1
		if l > 3 && args[3].Type == VAR_NUMBER {
			n = int(args[3].Value.(float64))
		}

		if reg, err := regexp.Compile(args[1].Value.(string)); err == nil {
			if n == -1 {
				return MakeVariable(VAR_STRING, reg.ReplaceAllString(args[0].Value.(string), args[2].Value.(string)))
			} else {
				return MakeVariable(VAR_STRING, strings.Join(reg.Split(args[0].Value.(string), n+1), args[2].Value.(string)))
			}
		}
	}

	return MakeVariable(VAR_STRING, "")
}

func regex_Split(this *Variable, args []Variable) Variable {
	var r []Variable

	if l := len(args); l > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		n := -1
		if l > 2 && args[2].Type == VAR_NUMBER {
			n = int(args[2].Value.(float64))
		}

		if reg, err := regexp.Compile(args[1].Value.(string)); err == nil {
			if n != -1 {
				n++
			}

			for _, s := range reg.Split(args[0].Value.(string), n) {
				r = append(r, MakeVariable(VAR_STRING, s))
			}
		}
	}

	return MakeVariable(VAR_ARRAY, &r)
}

/***********************************************/
/*                     math                    */
/***********************************************/
func math_Abs(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Abs(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Acos(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Acos(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Acosh(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Acosh(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Asin(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Asin(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Asinh(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Asinh(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Atan(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Atan(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Atanh(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Atanh(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Atan2(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Atan2(args[0].Value.(float64), args[1].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Cbrt(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Cbrt(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Ceil(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Ceil(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Cos(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Cos(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Cosh(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Cosh(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Exp(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Exp(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Expm1(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Expm1(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Floor(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Floor(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Hypot(this *Variable, args []Variable) Variable {
	if len(args) > 0 {
		var s float64

		for _, a := range args {
			if a.Type == VAR_NUMBER {
				s += math.Pow(a.Value.(float64), 2)
			} else {
				return MakeVariable(VAR_NUMBER, float64(0))
			}
		}

		return MakeVariable(VAR_NUMBER, math.Sqrt(s))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Log(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Log(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Log1p(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Log1p(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Log10(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Log10(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Log2(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Log2(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Max(this *Variable, args []Variable) Variable {
	r := -math.MaxFloat64

	for _, a := range args {
		if a.Type == VAR_NUMBER {
			if v := a.Value.(float64); v > r {
				r = v
			}
		}
	}

	return MakeVariable(VAR_NUMBER, r)
}

func math_Min(this *Variable, args []Variable) Variable {
	r := math.MaxFloat64

	for _, a := range args {
		if a.Type == VAR_NUMBER {
			if v := a.Value.(float64); v < r {
				r = v
			}
		}
	}

	return MakeVariable(VAR_NUMBER, r)
}

func math_Pow(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Pow(args[0].Value.(float64), args[1].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Random(this *Variable, args []Variable) Variable {
	return MakeVariable(VAR_NUMBER, rand.Float64())
}

func math_Round(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Round(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Sign(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		if v := args[0].Value.(float64); v < 0 {
			return MakeVariable(VAR_NUMBER, float64(-1))
		} else if v > 0 {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Sin(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Sin(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Sinh(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Sinh(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Sqrt(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Sqrt(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Tan(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Tan(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Tanh(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Tanh(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func math_Trunc(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, math.Trunc(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                      json                   */
/***********************************************/
func Escape(s string) string {
	return strings.Replace(strings.Replace(s, `\`, `\\`, -1), `"`, `\"`, -1)
}

func StringifyVariable(variable Variable) string {
	switch variable.Type {
	case VAR_VARIABLE:
		return StringifyVariable(*variable.Value.(*Variable))
	case VAR_NUMBER:
		return strconv.FormatFloat(variable.Value.(float64), 'f', -1, 64)
	case VAR_STRING:
		return `"` + Escape(variable.Value.(string)) + `"`
	case VAR_FUNCTION:
		return fmt.Sprint(&variable.Value)
	case VAR_ARRAY:
		{
			s := "["
			for i, e := range *variable.Value.(*[]Variable) {
				if i > 0 {
					s += ","
				}

				s += StringifyVariable(e)
			}
			return s + "]"
		}
	case VAR_OBJECT:
		{
			s := "{"
			f := false
			for k, v := range *variable.Value.(*map[string]*Variable) {
				if f {
					s += ","
				} else {
					f = true
				}

				s += `"` + Escape(k) + `"` + ":" + StringifyVariable(*v)
			}
			return s + "}"
		}
	}

	return fmt.Sprint(variable.Value)
}

func json_Stringify(this *Variable, args []Variable) Variable {
	if len(args) > 0 {
		return MakeVariable(VAR_STRING, StringifyVariable(args[0]))
	}

	return MakeVariable(VAR_STRING, "")
}

func ParseVariable(v interface{}) Variable {
	switch v.(type) {
	case uint:
		return MakeVariable(VAR_NUMBER, float64(v.(uint)))
	case uint8:
		return MakeVariable(VAR_NUMBER, float64(v.(uint8)))
	case uint16:
		return MakeVariable(VAR_NUMBER, float64(v.(uint16)))
	case uint32:
		return MakeVariable(VAR_NUMBER, float64(v.(uint32)))
	case uint64:
		return MakeVariable(VAR_NUMBER, float64(v.(uint64)))
	case int:
		return MakeVariable(VAR_NUMBER, float64(v.(int)))
	case int8:
		return MakeVariable(VAR_NUMBER, float64(v.(int8)))
	case int16:
		return MakeVariable(VAR_NUMBER, float64(v.(int16)))
	case int32:
		return MakeVariable(VAR_NUMBER, float64(v.(int32)))
	case int64:
		return MakeVariable(VAR_NUMBER, float64(v.(int64)))
	case float32:
		return MakeVariable(VAR_NUMBER, float64(v.(float32)))
	case float64:
		return MakeVariable(VAR_NUMBER, float64(v.(float64)))
	case string:
		return MakeVariable(VAR_STRING, v.(string))
	case map[string]interface{}:
		r := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
		for k, e := range v.(map[string]interface{}) {
			AddProp(&r, k, ParseVariable(e))
		}
		return r
	case []interface{}:
		var r []Variable
		for _, e := range v.([]interface{}) {
			r = append(r, ParseVariable(e))
		}
		return MakeVariable(VAR_ARRAY, &r)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func json_Parse(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_STRING {
		var v interface{}
		err := json.Unmarshal([]byte(args[0].Value.(string)), &v)
		if err == nil {
			return ParseVariable(v)
		}
	}

	return MakeVariable(VAR_OBJECT, &map[string]*Variable{})
}

/***********************************************/
/*                      http                   */
/***********************************************/
func http_Get(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_STRING {
		resp, err := http.Get(args[0].Value.(string))
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				return MakeVariable(VAR_STRING, string(body))
			}
		}
	}

	return MakeVariable(VAR_STRING, "")
}

func http_Request(this *Variable, args []Variable) Variable {
	ret := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if len(args) > 0 && args[0].Type == VAR_OBJECT {
		obj := *args[0].Value.(*map[string]*Variable)
		url := GetProp(obj, "URL")
		method := GetProp(obj, "Method")
		obody := GetProp(obj, "Body")
		headers := GetProp(obj, "Headers")

		if url.Type == VAR_STRING && method.Type == VAR_STRING {
			var data io.Reader
			if obody.Type == VAR_STRING {
				data = bytes.NewBuffer([]byte(obody.Value.(string)))
			}

			req, err := http.NewRequest(method.Value.(string), url.Value.(string), data)
			if err != nil {
				return MakeVariable(VAR_STRING, err.Error())
			}

			if headers.Type == VAR_OBJECT {
				for k, v := range *headers.Value.(*map[string]*Variable) {
					req.Header.Set(k, ToString(*v))
				}
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return MakeVariable(VAR_STRING, err.Error())
			}

			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return MakeVariable(VAR_STRING, err.Error())
			}

			AddProp(&ret, "Status", MakeVariable(VAR_NUMBER, float64(resp.StatusCode)))
			AddProp(&ret, "Body", MakeVariable(VAR_STRING, string(body)))
		}
	}

	return ret
}

/***********************************************/
/*                     thread                  */
/***********************************************/
func thread_Sleep(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_NUMBER {
		time.Sleep(time.Millisecond * time.Duration(args[0].Value.(float64)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func thread_Create(this *Variable, args []Variable) Variable {
	if len(args) > 0 {
		if v := args[0]; v.Type == VAR_FUNCTION {
			args = args[1:]
			f := v.Value.([]Tree)

			var thread []Stack
			stack := StackAdd(&thread, -1)

			StackPush("arguments", &thread, stack, MakeVariable(VAR_ARRAY, &args))
			for i, e := range f[0].C {
				if i < len(args) {
					StackPush(e.T.Value.(string), &thread, stack, args[i])
				} else {
					StackPush(e.T.Value.(string), &thread, stack, MakeVariable(VAR_NUMBER, float64(0)))
				}
			}

			go func() {
				for _, e := range f[1].C {
					if v := Eval(e, &thread, stack); v.Type == VAR_RETURN {
						break
					}
				}

				StackRemove(&thread)
			}()
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                     winapi                  */
/***********************************************/
var procQueryFullProcessImageNameW, procCreateThread, procEnumWindows, procGetParent, procVirtualProtectEx, procCreateRemoteThread, procGetCurrentProcessId, procIsWow64Process, procReadProcessMemory, procWriteProcessMemory, procMapVirtualKey, procVkKeyScanA, procTimeGetTime, procNtSuspendProcess, procNtResumeProcess, procVirtualQueryEx, procOpenThread, procSuspendThread, procResumeThread, procThread32First, procThread32Next, procWow64GetThreadContext, procWow64GetThreadSelectorEntry, procGetThreadTimes, procNtQueryInformationThread, procVirtualAllocEx *syscall.Proc

type MEMORY_BASIC_INFORMATION struct {
	BaseAddress       w32.PVOID
	AllocationBase    w32.PVOID
	AllocationProtect w32.DWORD
	RegionSize        w32.SIZE_T
	State             w32.DWORD
	Protect           w32.DWORD
	Type              w32.DWORD
}

type THREADENTRY32 struct {
	DwSize             w32.DWORD
	CntUsage           w32.DWORD
	Th32ThreadID       w32.DWORD
	Th32OwnerProcessID w32.DWORD
	TpBasePri          w32.DWORD
	TpDeltaPri         w32.DWORD
	DwFlags            w32.DWORD
}

type WOW64_FLOATING_SAVE_AREA struct {
	ControlWord   w32.DWORD
	StatusWord    w32.DWORD
	TagWord       w32.DWORD
	ErrorOffset   w32.DWORD
	ErrorSelector w32.DWORD
	DataOffset    w32.DWORD
	DataSelector  w32.DWORD
	RegisterArea  [80]byte
	Cr0NpxState   w32.DWORD
}

type WOW64_CONTEXT struct {
	ContextFlags      w32.DWORD
	Dr0               w32.DWORD
	Dr1               w32.DWORD
	Dr2               w32.DWORD
	Dr3               w32.DWORD
	Dr6               w32.DWORD
	Dr7               w32.DWORD
	FloatSave         WOW64_FLOATING_SAVE_AREA
	SegGs             w32.DWORD
	SegFs             w32.DWORD
	SegEs             w32.DWORD
	SegDs             w32.DWORD
	Edi               w32.DWORD
	Esi               w32.DWORD
	Ebx               w32.DWORD
	Edx               w32.DWORD
	Ecx               w32.DWORD
	Eax               w32.DWORD
	Ebp               w32.DWORD
	Eip               w32.DWORD
	SegCs             w32.DWORD
	EFlags            w32.DWORD
	Esp               w32.DWORD
	SegSs             w32.DWORD
	ExtendedRegisters [512]byte
}

type WOW64_LDT_ENTRY struct {
	LimitLow uint16
	BaseLow  uint16
	BaseMid  byte
	Flags1   byte
	Flags2   byte
	BaseHi   byte
}

type CLIENT_ID struct {
	UniqueProcess w32.PVOID
	UniqueThread  w32.PVOID
}

type THREAD_BASIC_INFORMATION struct {
	ExitStatus     w32.DWORD
	TebBaseAddress w32.PVOID
	ClientId       CLIENT_ID
	AffinityMask   w32.ULONG_PTR
	Priority       w32.DWORD
	BasePriority   w32.DWORD
}

func IsErrSuccess(err error) bool {
	if e, ok := err.(syscall.Errno); ok {
		if e == 0 {
			return true
		}
	}

	return false
}

func QueryFullProcessImageNameW(h w32.HANDLE) string {
	size := uint32(0xFFF)
	path := make([]uint16, size)
	procQueryFullProcessImageNameW.Call(uintptr(h), uintptr(0), uintptr(unsafe.Pointer(&path[0])), uintptr(unsafe.Pointer(&size)))
	return windows.UTF16ToString(path)
}

func CreateThread(callback uintptr, param uintptr) w32.HANDLE {
	handle, _, err := procCreateThread.Call(uintptr(0), uintptr(0), callback, param, uintptr(0), uintptr(0))
	if !IsErrSuccess(err) {
		return 0
	}

	return w32.HANDLE(handle)
}

func EnumWindows(callback uintptr, param uintptr) bool {
	if _, _, err := procEnumWindows.Call(callback, param); IsErrSuccess(err) {
		return true
	}

	return false
}

func VirtualProtectEx(h w32.HANDLE, addr uintptr, size uint32, protect uint32) uint32 {
	old := uint32(0)
	if _, _, err := procVirtualProtectEx.Call(uintptr(h), addr, uintptr(size), uintptr(protect), uintptr(unsafe.Pointer(&old))); !IsErrSuccess(err) {
		return 0
	}

	return old
}

func GetCurrentProcessId() uint {
	r, _, err := procGetCurrentProcessId.Call()
	if !IsErrSuccess(err) {
		return 0
	}

	return uint(r)
}

func IsWow64Process(hProcess w32.HANDLE, out *bool) {
	procIsWow64Process.Call(uintptr(hProcess), uintptr(unsafe.Pointer(out)))
}

func ReadProcessMemory(hProcess w32.HANDLE, lpBaseAddress uintptr, size uint) (data []byte, err error) {
	data = make([]byte, size)
	read := uint(0)
	_, _, err = procReadProcessMemory.Call(uintptr(hProcess), lpBaseAddress, uintptr(unsafe.Pointer(&data[0])), uintptr(size), uintptr(unsafe.Pointer(&read)))
	if !IsErrSuccess(err) {
		return
	}

	err = nil

	return
}

func WriteProcessMemory(hProcess w32.HANDLE, lpBaseAddress uintptr, data uintptr, size uint) (err error) {
	_, _, err = procWriteProcessMemory.Call(uintptr(hProcess), lpBaseAddress, data, uintptr(size), uintptr(0))
	if !IsErrSuccess(err) {
		return
	}

	err = nil
	return
}

func MapVirtualKey(k uint, t uint) uint16 {
	r, _, err := procMapVirtualKey.Call(uintptr(k), uintptr(t))
	if !IsErrSuccess(err) {
		return 0
	}

	return uint16(r)
}

func VkKeyScanA(k byte) uint16 {
	r, _, err := procVkKeyScanA.Call(uintptr(k))
	if !IsErrSuccess(err) {
		return 0
	}

	return uint16(r)
}

func GetParent(hwnd w32.HWND) w32.HWND {
	r, _, err := procGetParent.Call(uintptr(hwnd))
	if !IsErrSuccess(err) {
		return 0
	}

	return w32.HWND(r)
}

func TimeGetTime() uint {
	r, _, err := procTimeGetTime.Call()
	if !IsErrSuccess(err) {
		return 0
	}

	return uint(r)
}

func NtSuspendProcess(h w32.HANDLE) {
	procNtSuspendProcess.Call(uintptr(h))
}

func NtResumeProcess(h w32.HANDLE) {
	procNtResumeProcess.Call(uintptr(h))
}

func VirtualQueryEx(h w32.HANDLE, a w32.LPCVOID, b uintptr, l w32.SIZE_T) int {
	r, _, err := procVirtualQueryEx.Call(uintptr(h), uintptr(a), b, uintptr(l))
	if !IsErrSuccess(err) {
		return 0
	}

	return int(w32.SIZE_T(r))
}

func OpenThread(a w32.DWORD, i w32.BOOL, id w32.DWORD) w32.HANDLE {
	r, _, err := procOpenThread.Call(uintptr(a), uintptr(i), uintptr(id))
	if !IsErrSuccess(err) {
		return 0
	}

	return w32.HANDLE(r)
}

func SuspendThread(h w32.HANDLE) bool {
	_, _, err := procSuspendThread.Call(uintptr(h))
	if !IsErrSuccess(err) {
		return false
	}

	return true
}

func ResumeThread(h w32.HANDLE) bool {
	_, _, err := procResumeThread.Call(uintptr(h))
	if !IsErrSuccess(err) {
		return false
	}

	return true
}

func Thread32First(h w32.HANDLE, te *THREADENTRY32) bool {
	r, _, err := procThread32First.Call(uintptr(h), uintptr(unsafe.Pointer(te)))
	if !IsErrSuccess(err) {
		return false
	}

	return int(r) > 0
}

func Thread32Next(h w32.HANDLE, te *THREADENTRY32) bool {
	r, _, err := procThread32Next.Call(uintptr(h), uintptr(unsafe.Pointer(te)))
	if !IsErrSuccess(err) {
		return false
	}

	return int(r) > 0
}

func Wow64GetThreadContext(h w32.HANDLE, pc *WOW64_CONTEXT) bool {
	r, _, err := procWow64GetThreadContext.Call(uintptr(h), uintptr(unsafe.Pointer(pc)))
	if !IsErrSuccess(err) {
		return false
	}

	return int(r) > 0
}

func Wow64GetThreadSelectorEntry(h w32.HANDLE, s w32.DWORD, se *WOW64_LDT_ENTRY) bool {
	r, _, err := procWow64GetThreadSelectorEntry.Call(uintptr(h), uintptr(s), uintptr(unsafe.Pointer(se)))
	if !IsErrSuccess(err) {
		return false
	}

	return int(r) > 0
}

func GetThreadCreationTime(h w32.HANDLE) int64 {
	var filetime, idle windows.Filetime
	pidle := uintptr(unsafe.Pointer(&idle))
	r, _, err := procGetThreadTimes.Call(uintptr(h), uintptr(unsafe.Pointer(&filetime)), pidle, pidle, pidle)
	if !IsErrSuccess(err) || r == 0 {
		return 0
	}

	return filetime.Nanoseconds()
}

func NtQueryInformationThread(h w32.HANDLE, ic int32, ti *THREAD_BASIC_INFORMATION) {
	procNtQueryInformationThread.Call(uintptr(h), uintptr(ic), uintptr(unsafe.Pointer(ti)), uintptr(unsafe.Sizeof(*ti)), uintptr(0))
}

func GetModuleInfoByName(pid uint32, name string) w32.MODULEENTRY32 {
	snapshot := w32.CreateToolhelp32Snapshot(w32.TH32CS_SNAPMODULE|w32.TH32CS_SNAPMODULE32, pid)
	if snapshot != 0 {
		defer w32.CloseHandle(snapshot)

		var entry w32.MODULEENTRY32
		entry.Size = uint32(unsafe.Sizeof(entry))

		if w32.Module32First(snapshot, &entry) {
			c := true
			for c {
				if strings.EqualFold(windows.UTF16ToString(entry.SzModule[:]), name) {
					return entry
				}

				c = w32.Module32Next(snapshot, &entry)
			}
		}
	}

	return w32.MODULEENTRY32{}
}

func CreateRemoteThread(h w32.HANDLE, addr uintptr, arg uintptr) w32.HANDLE {
	r, _, err := procCreateRemoteThread.Call(uintptr(h), uintptr(0), uintptr(0), addr, arg, uintptr(0), uintptr(0))
	if !IsErrSuccess(err) {
		return 0
	}

	return w32.HANDLE(r)
}

func VirtualAllocEx(h w32.HANDLE, addr uintptr, size uint32, alloc_type uint32, protect uint32) uintptr {
	r, _, _ := procVirtualAllocEx.Call(uintptr(h), addr, uintptr(size), uintptr(alloc_type), uintptr(protect))
	return r
}

/***********************************************/
/*                      date                   */
/***********************************************/
func date_Now(this *Variable, args []Variable) Variable {
	return MakeVariable(VAR_NUMBER, float64(TimeGetTime()))
}

func date_Time(this *Variable, args []Variable) Variable {
	d := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if time, err := w32.GetSystemTime(); err == nil {
		AddProp(&d, "Milliseconds", MakeVariable(VAR_NUMBER, float64(time.Milliseconds)))
		AddProp(&d, "Second", MakeVariable(VAR_NUMBER, float64(time.Second)))
		AddProp(&d, "Minute", MakeVariable(VAR_NUMBER, float64(time.Minute)))
		AddProp(&d, "Hour", MakeVariable(VAR_NUMBER, float64(time.Hour)))
		AddProp(&d, "Day", MakeVariable(VAR_NUMBER, float64(time.Day)))
		AddProp(&d, "DayOfWeek", MakeVariable(VAR_NUMBER, float64(time.DayOfWeek)))
		AddProp(&d, "Month", MakeVariable(VAR_NUMBER, float64(time.Month)))
		AddProp(&d, "Year", MakeVariable(VAR_NUMBER, float64(time.Year)))
	}

	return d
}

/***********************************************/
/*                      file                   */
/***********************************************/
func FileObject(file *os.File) Variable {
	obj := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&obj, "Close", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		file.Close()
		return MakeVariable(VAR_NUMBER, float64(0))
	}))
	AddProp(&obj, "Seek", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		if len(args) > 0 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
			if n, err := file.Seek(int64(args[0].Value.(float64)), int(args[1].Value.(float64))); err == nil {
				return MakeVariable(VAR_NUMBER, float64(n))
			}
		}
		return MakeVariable(VAR_NUMBER, float64(0))
	}))
	AddProp(&obj, "Read", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		var r []Variable
		if len(args) > 0 && args[0].Type == VAR_NUMBER {
			b := make([]byte, int(args[0].Value.(float64)))
			if n, err := file.Read(b); err == nil || err == io.EOF {
				for i := 0; i < n; i++ {
					r = append(r, MakeVariable(VAR_NUMBER, float64(b[i])))
				}
			}
		}
		return MakeVariable(VAR_ARRAY, &r)
	}))
	AddProp(&obj, "ReadAt", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		var r []Variable
		if len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
			b := make([]byte, int(args[0].Value.(float64)))
			if n, err := file.ReadAt(b, int64(args[1].Value.(float64))); err == nil || err == io.EOF {
				for i := 0; i < n; i++ {
					r = append(r, MakeVariable(VAR_NUMBER, float64(b[i])))
				}
			}
		}
		return MakeVariable(VAR_ARRAY, &r)
	}))
	AddProp(&obj, "Write", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		if len(args) > 0 {
			if args[0].Type == VAR_ARRAY {
				var buffer []byte
				for _, b := range *args[0].Value.(*[]Variable) {
					if b.Type != VAR_NUMBER {
						return MakeVariable(VAR_NUMBER, float64(0))
					}
					buffer = append(buffer, byte(b.Value.(float64)))
				}

				if n, err := file.Write(buffer); err == nil || err == io.EOF {
					return MakeVariable(VAR_NUMBER, float64(n))
				}
			} else if args[0].Type == VAR_STRING {
				if n, err := file.WriteString(args[0].Value.(string)); err == nil {
					return MakeVariable(VAR_NUMBER, float64(n))
				}
			}
		}
		return MakeVariable(VAR_NUMBER, float64(0))
	}))
	AddProp(&obj, "WriteAt", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		if len(args) > 1 && args[1].Type == VAR_NUMBER {
			var buffer []byte
			if args[0].Type == VAR_ARRAY {
				for _, b := range *args[0].Value.(*[]Variable) {
					if b.Type != VAR_NUMBER {
						return MakeVariable(VAR_NUMBER, float64(0))
					}
					buffer = append(buffer, byte(b.Value.(float64)))
				}
			} else if args[1].Type == VAR_STRING {
				buffer = []byte(args[0].Value.(string))
			}

			if n, err := file.WriteAt(buffer, int64(args[1].Value.(float64))); err == nil || err == io.EOF {
				return MakeVariable(VAR_NUMBER, float64(n))
			}
		}
		return MakeVariable(VAR_NUMBER, float64(0))
	}))
	AddProp(&obj, "Stat", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
		r := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
		if info, err := file.Stat(); err == nil {
			AddProp(&r, "Name", MakeVariable(VAR_STRING, info.Name()))
			AddProp(&r, "Size", MakeVariable(VAR_NUMBER, float64(info.Size())))
			AddProp(&r, "Mode", MakeVariable(VAR_NUMBER, float64(info.Mode())))
			AddProp(&r, "ModTime", MakeVariable(VAR_NUMBER, float64(info.ModTime().UnixNano())))
			if info.IsDir() {
				AddProp(&r, "IsDir", MakeVariable(VAR_NUMBER, float64(1)))
			} else {
				AddProp(&r, "IsDir", MakeVariable(VAR_NUMBER, float64(0)))
			}
		}
		return r
	}))
	return obj
}

func file_Open(this *Variable, args []Variable) Variable {
	if l := len(args); l > 0 && args[0].Type == VAR_STRING {
		flag := os.O_RDWR | os.O_CREATE
		perm := 0777
		if l > 1 && args[1].Type == VAR_NUMBER {
			flag = int(args[1].Value.(float64))
			if l > 2 && args[2].Type == VAR_NUMBER {
				perm = int(args[2].Value.(float64))
			}
		}

		if file, err := os.OpenFile(args[0].Value.(string), flag, os.FileMode(perm)); err == nil {
			return FileObject(file)
		}
	}

	return MakeVariable(VAR_OBJECT, &map[string]*Variable{})
}

func file_Remove(this *Variable, args []Variable) Variable {
	if l := len(args); l > 0 && args[0].Type == VAR_STRING {
		if err := os.Remove(args[0].Value.(string)); err != nil {
			return MakeVariable(VAR_STRING, (err.(*os.PathError)).Path)
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func file_RemoveAll(this *Variable, args []Variable) Variable {
	if l := len(args); l > 0 && args[0].Type == VAR_STRING {
		if err := os.RemoveAll(args[0].Value.(string)); err != nil {
			return MakeVariable(VAR_STRING, (err.(*os.PathError)).Path)
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                    process                  */
/***********************************************/
func process_Open(this *Variable, args []Variable) Variable {
	p := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	id := -1
	handle := w32.HANDLE(0)
	wow64 := false

	if len(args) > 0 {
		var comp func(windows.ProcessEntry32) bool
		if v := args[0]; v.Type == VAR_STRING {
			comp = func(entry windows.ProcessEntry32) bool {
				return strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), v.Value.(string))
			}
		} else if v.Type == VAR_NUMBER {
			comp = func(entry windows.ProcessEntry32) bool {
				return entry.ProcessID == uint32(v.Value.(float64))
			}
		}

		if comp != nil {
			// Use a snapshot even if given the PID to get extra information on the process
			snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
			if err == nil {
				var entry windows.ProcessEntry32
				entry.Size = uint32(unsafe.Sizeof(entry))

				if windows.Process32First(snapshot, &entry) == nil {
					var c error
					for c == nil {
						if comp(entry) {
							id = int(entry.ProcessID)
							AddProp(&p, "ParentId", MakeVariable(VAR_NUMBER, float64(entry.ParentProcessID)))
							AddProp(&p, "PriClassBase", MakeVariable(VAR_NUMBER, float64(entry.PriClassBase)))
							AddProp(&p, "Name", MakeVariable(VAR_STRING, windows.UTF16ToString(entry.ExeFile[:])))
							break
						}

						c = windows.Process32Next(snapshot, &entry)
					}
				}

				windows.CloseHandle(snapshot)
			}

			if id != -1 {
				handle, _ = w32.OpenProcess(w32.PROCESS_ALL_ACCESS, false, uint32(id))
				if handle != 0 {
					IsWow64Process(handle, &wow64)
				}
			} else {
				return p
			}
		}
	}

	AddProp(&p, "Handle", MakeVariable(VAR_NUMBER, float64(handle)))
	AddProp(&p, "Id", MakeVariable(VAR_NUMBER, float64(id)))
	AddProp(&p, "Path", MakeVariable(VAR_STRING, QueryFullProcessImageNameW(handle)))
	AddProp(&p, "Suspend", MakeVariable(VAR_NFUNCTION, process_Suspend))
	AddProp(&p, "Resume", MakeVariable(VAR_NFUNCTION, process_Resume))
	if wow64 {
		AddProp(&p, "Wow64", MakeVariable(VAR_NUMBER, float64(1)))
		AddProp(&p, "GetProcAddress", MakeVariable(VAR_NFUNCTION, process_GetProcAddress32))
		AddProp(&p, "Call", MakeVariable(VAR_NFUNCTION, process_Call32))
		AddProp(&p, "Hook", MakeVariable(VAR_NFUNCTION, process_Hook32))
		AddProp(&p, "ReadPointer", MakeVariable(VAR_NFUNCTION, process_ReadPointer32))
		AddProp(&p, "LoadLibrary", MakeVariable(VAR_NFUNCTION, process_LoadLibrary32))
	} else {
		AddProp(&p, "Wow64", MakeVariable(VAR_NUMBER, float64(0)))
		AddProp(&p, "GetProcAddress", MakeVariable(VAR_NFUNCTION, process_GetProcAddress64))
		AddProp(&p, "Call", MakeVariable(VAR_NFUNCTION, process_Call64))
		AddProp(&p, "Hook", MakeVariable(VAR_NFUNCTION, process_Hook64))
		AddProp(&p, "ReadPointer", MakeVariable(VAR_NFUNCTION, process_ReadPointer64))
		AddProp(&p, "LoadLibrary", MakeVariable(VAR_NFUNCTION, process_LoadLibrary64))
	}
	AddProp(&p, "FindPattern", MakeVariable(VAR_NFUNCTION, process_FindPattern))
	AddProp(&p, "Protect", MakeVariable(VAR_NFUNCTION, process_Protect))
	AddProp(&p, "Alloc", MakeVariable(VAR_NFUNCTION, process_Alloc))
	AddProp(&p, "Free", MakeVariable(VAR_NFUNCTION, process_Free))
	AddProp(&p, "Read", MakeVariable(VAR_NFUNCTION, process_Read))
	AddProp(&p, "Write", MakeVariable(VAR_NFUNCTION, process_Write))
	AddProp(&p, "ReadString", MakeVariable(VAR_NFUNCTION, process_ReadString))
	AddProp(&p, "ReadString8", MakeVariable(VAR_NFUNCTION, process_ReadString))
	AddProp(&p, "WriteString", MakeVariable(VAR_NFUNCTION, process_WriteString))
	AddProp(&p, "WriteString8", MakeVariable(VAR_NFUNCTION, process_WriteString))
	AddProp(&p, "ReadString16", MakeVariable(VAR_NFUNCTION, process_ReadString16))
	AddProp(&p, "WriteString16", MakeVariable(VAR_NFUNCTION, process_WriteString16))
	AddProp(&p, "ReadInt16", MakeVariable(VAR_NFUNCTION, process_ReadInt16))
	AddProp(&p, "WriteInt16", MakeVariable(VAR_NFUNCTION, process_WriteInt16))
	AddProp(&p, "ReadInt", MakeVariable(VAR_NFUNCTION, process_ReadInt32))
	AddProp(&p, "WriteInt", MakeVariable(VAR_NFUNCTION, process_WriteInt32))
	AddProp(&p, "ReadInt32", MakeVariable(VAR_NFUNCTION, process_ReadInt32))
	AddProp(&p, "WriteInt32", MakeVariable(VAR_NFUNCTION, process_WriteInt32))
	AddProp(&p, "ReadInt64", MakeVariable(VAR_NFUNCTION, process_ReadInt64))
	AddProp(&p, "WriteInt64", MakeVariable(VAR_NFUNCTION, process_WriteInt64))
	AddProp(&p, "ReadUint16", MakeVariable(VAR_NFUNCTION, process_ReadUint16))
	AddProp(&p, "WriteUint16", MakeVariable(VAR_NFUNCTION, process_WriteUint16))
	AddProp(&p, "ReadUint", MakeVariable(VAR_NFUNCTION, process_ReadUint32))
	AddProp(&p, "WriteUint", MakeVariable(VAR_NFUNCTION, process_WriteUint32))
	AddProp(&p, "ReadUint32", MakeVariable(VAR_NFUNCTION, process_ReadUint32))
	AddProp(&p, "WriteUint32", MakeVariable(VAR_NFUNCTION, process_WriteUint32))
	AddProp(&p, "ReadUint64", MakeVariable(VAR_NFUNCTION, process_ReadUint64))
	AddProp(&p, "WriteUint64", MakeVariable(VAR_NFUNCTION, process_WriteUint64))
	AddProp(&p, "ReadFloat", MakeVariable(VAR_NFUNCTION, process_ReadFloat))
	AddProp(&p, "WriteFloat", MakeVariable(VAR_NFUNCTION, process_WriteFloat))
	AddProp(&p, "ReadFloat32", MakeVariable(VAR_NFUNCTION, process_ReadFloat))
	AddProp(&p, "WriteFloat32", MakeVariable(VAR_NFUNCTION, process_WriteFloat))
	AddProp(&p, "ReadFloat64", MakeVariable(VAR_NFUNCTION, process_ReadDouble))
	AddProp(&p, "WriteFloat64", MakeVariable(VAR_NFUNCTION, process_WriteDouble))
	AddProp(&p, "ReadDouble", MakeVariable(VAR_NFUNCTION, process_ReadDouble))
	AddProp(&p, "WriteDouble", MakeVariable(VAR_NFUNCTION, process_WriteDouble))
	AddProp(&p, "Modules", MakeVariable(VAR_NFUNCTION, process_Modules))
	AddProp(&p, "Windows", MakeVariable(VAR_NFUNCTION, process_Windows))
	AddProp(&p, "Threads", MakeVariable(VAR_NFUNCTION, process_Threads))
	AddProp(&p, "Close", MakeVariable(VAR_NFUNCTION, process_Close))
	AddProp(&p, "Exit", MakeVariable(VAR_NFUNCTION, process_Exit))
	return p
}

func process_List(this *Variable, args []Variable) Variable {
	var r []Variable

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err == nil {
		var entry windows.ProcessEntry32
		entry.Size = uint32(unsafe.Sizeof(entry))

		if windows.Process32First(snapshot, &entry) == nil {
			var c error
			for c == nil {
				p := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
				AddProp(&p, "Id", MakeVariable(VAR_NUMBER, float64(entry.ProcessID)))
				AddProp(&p, "Threads", MakeVariable(VAR_NUMBER, float64(entry.Threads)))
				AddProp(&p, "ParentId", MakeVariable(VAR_NUMBER, float64(entry.ParentProcessID)))
				AddProp(&p, "PriClassBase", MakeVariable(VAR_NUMBER, float64(entry.PriClassBase)))
				AddProp(&p, "Name", MakeVariable(VAR_STRING, windows.UTF16ToString(entry.ExeFile[:])))
				r = append(r, p)
				c = windows.Process32Next(snapshot, &entry)
			}
		}

		windows.CloseHandle(snapshot)
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func GetHandle(this *Variable) (handle w32.HANDLE) {
	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Handle"]; ok && v.Type == VAR_NUMBER {
				handle = w32.HANDLE(v.Value.(float64))
			}
		}
	}

	return
}

func GetId(this *Variable) (id uint32) {
	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Id"]; ok && v.Type == VAR_NUMBER {
				id = uint32(v.Value.(float64))
			}
		}
	}

	return
}

func PushArgs32(buffer *[]byte, args []interface{}) (pops int) {
	var b []byte

	for i := len(args) - 1; i > -1; i-- {
		pops++

		switch args[i].(type) {
		case uint8:
			b = append(b, 0x6A)
			b = append(b, args[i].(uint8))
		case uint16:
			b = append(b, 0x68)
			v := uint32(args[i].(uint16))
			for _, e := range *(*[4]byte)(unsafe.Pointer(&v)) {
				b = append(b, e)
			}
		case uint32:
			b = append(b, 0x68)
			v := args[i].(uint32)
			for _, e := range *(*[4]byte)(unsafe.Pointer(&v)) {
				b = append(b, e)
			}
		case uint64:
			b = append(b, 0x68)

			v := args[i].(uint64)
			bytes := *(*[8]byte)(unsafe.Pointer(&v))
			for i := 4; i < 8; i++ {
				b = append(b, bytes[i])
			}

			b = append(b, 0x68)

			for i := 0; i < 4; i++ {
				b = append(b, bytes[i])
			}

			pops++
		case float32:
			b = append(b, 0x68)
			v := args[i].(float32)
			for _, e := range *(*[4]byte)(unsafe.Pointer(&v)) {
				b = append(b, e)
			}
		case float64:
			b = append(b, 0x68)

			v := args[i].(float64)
			bytes := *(*[8]byte)(unsafe.Pointer(&v))
			for i := 4; i < 8; i++ {
				b = append(b, bytes[i])
			}

			b = append(b, 0x68)

			for i := 0; i < 4; i++ {
				b = append(b, bytes[i])
			}

			pops++
		}
	}

	*buffer = append(*buffer, b...)
	return
}

func AbsCall32(buffer *[]byte, addr uint32) {
	b := []byte{0xB8} // mov eax,x
	for _, e := range *(*[4]byte)(unsafe.Pointer(&addr)) {
		b = append(b, e)
	}
	*buffer = append(*buffer, append(b, []byte{0xFF, 0xD0}...)...) // call eax
}

func Call32(h w32.HANDLE, addr uint32, flags uint32, args ...interface{}) (raw_ret []byte) {
	var buffer []byte

	if flags&0x20 != 0 { // stdcall
		PushArgs32(&buffer, args)
		AbsCall32(&buffer, addr)
	} else if flags&0x40 != 0 { // fastcall - ecx(index 0 dword or less), edx(index 1 dword or less), rest stack
		regs := []byte{0xB9, 0xBA}
		state := 0
		i := 0

		AddFastReg := func(v uint32) {
			buffer = append(buffer, regs[state])
			for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
				buffer = append(buffer, b)
			}
			args = append(args[:i], args[i+1:]...)
			i--
			state++
		}

		for ; i < len(args) && state < 2; i++ {
			switch args[i].(type) {
			case uint8:
				AddFastReg(uint32(args[i].(uint8)))
			case uint16:
				AddFastReg(uint32(args[i].(uint16)))
			case uint32:
				AddFastReg(args[i].(uint32))
			}
		}

		PushArgs32(&buffer, args)
		AbsCall32(&buffer, addr)
	} else if flags&0x80 != 0 { // thiscall - ecx(this)
		if len(args) > 0 {
			if v, ok := args[0].(uint32); ok {
				buffer = append(buffer, 0xB9)
				for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
				args = args[1:]
			} else {
				return
			}
		}

		PushArgs32(&buffer, args)
		AbsCall32(&buffer, addr)
	} else { // cdecl - caller cleanup
		pops := PushArgs32(&buffer, args)
		AbsCall32(&buffer, addr)
		for i := 0; i < pops; i++ {
			buffer = append(buffer, 0x59)
		}
	}

	if ret := uint32(VirtualAllocEx(h, 0, 8, w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE)); ret != 0 {
		if flags&0x01 != 0 { // int64
			buffer = append(buffer, 0xA3)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&ret)) {
				buffer = append(buffer, b)
			}

			v := ret + 4
			buffer = append(buffer, []byte{0x89, 0x15}...)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
				buffer = append(buffer, b)
			}
		} else if flags&0x02 != 0 { // float32
			buffer = append(buffer, []byte{0xD9, 0x1D}...)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&ret)) {
				buffer = append(buffer, b)
			}
		} else if flags&0x04 != 0 { // float64
			buffer = append(buffer, []byte{0xDD, 0x1D}...)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&ret)) {
				buffer = append(buffer, b)
			}
		} else { // int32
			buffer = append(buffer, 0xA3)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&ret)) {
				buffer = append(buffer, b)
			}
		}

		buffer = append(buffer, 0xC3)

		if proc := VirtualAllocEx(h, 0, uint32(len(buffer)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); proc != 0 && WriteProcessMemory(h, proc, uintptr(unsafe.Pointer(&buffer[0])), uint(len(buffer))) == nil {
			if thread := CreateRemoteThread(h, proc, 0); thread != 0 {
				if flags&0x10 == 0 {
					windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)

					if flags&0x01 != 0 || flags&0x04 != 0 {
						if bytes, err := ReadProcessMemory(h, uintptr(ret), 8); err == nil {
							for _, b := range bytes {
								raw_ret = append(raw_ret, b)
							}
						}
					} else {
						if bytes, err := ReadProcessMemory(h, uintptr(ret), 4); err == nil {
							for _, b := range bytes {
								raw_ret = append(raw_ret, b)
							}
						}
					}
				}

				w32.CloseHandle(thread)
			}

			w32.VirtualFreeEx(h, proc, 0, w32.MEM_RELEASE)
		}

		w32.VirtualFreeEx(h, uintptr(ret), 0, w32.MEM_RELEASE)
	}

	return
}

func Call64(h w32.HANDLE, addr uint64, flags uint32, args ...interface{}) (raw_ret [8]byte) {
	BYTE_REG_STUBS := [][]byte{{0xB1}, {0xB2}, {0x41, 0xB0}, {0x41, 0xB1}}                                                                                       // mov reg,x
	BYTE_STACK_STUB := []byte{0xC6, 0x44, 0x24}                                                                                                                  // mov byte ptr [rsp+x],y
	WORD_REG_STUBS := [][]byte{{0x66, 0xB9}, {0x66, 0xBA}, {0x66, 0x41, 0xB8}, {0x66, 0x41, 0xB9}}                                                               // mov reg,x
	WORD_STACK_STUB := []byte{0x66, 0xC7, 0x44, 0x24}                                                                                                            // mov word ptr [rsp+x],y
	DWORD_REG_STUBS := [][]byte{{0xB9}, {0xBA}, {0x41, 0xB8}, {0x41, 0xB9}}                                                                                      // mov reg,x
	DWORD_STACK_STUB := []byte{0xC7, 0x44, 0x24}                                                                                                                 // mov dword ptr [rsp+x],y
	QWORD_REG_STUBS := [][]byte{{0x48, 0xB9}, {0x48, 0xBA}, {0x49, 0xB8}, {0x49, 0xB9}}                                                                          // mov reg,x
	QWORD_STACK_STUB := []byte{0x48, 0x89, 0x44, 0x24}                                                                                                           // mov qword ptr [rsp+x],rax
	FLOAT_REG_STUBS := [][]byte{{0x66, 0x0F, 0x6E, 0xC0}, {0x66, 0x0F, 0x6E, 0xC8}, {0x66, 0x0F, 0x6E, 0xD0}, {0x66, 0x0F, 0x6E, 0xD8}}                          // mov reg,eax
	DOUBLE_REG_STUBS := [][]byte{{0x66, 0x48, 0x0F, 0x6E, 0xC0}, {0x66, 0x48, 0x0F, 0x6E, 0xC8}, {0x66, 0x48, 0x0F, 0x6E, 0xD0}, {0x66, 0x48, 0x0F, 0x6E, 0xD8}} // mov reg,rax

	shadow := byte(0x28)
	le := len(args)
	if le > 4 {
		shadow += byte(math.Ceil((float64(le)-4)/2) * 16)
	}

	buffer := []byte{0x48, 0x83, 0xEC, shadow}

	for i := le - 1; i > 3; i-- {
		switch args[i].(type) {
		case uint8:
			buffer = append(buffer, BYTE_STACK_STUB...)
			buffer = append(buffer, []byte{byte(i * 8), args[i].(uint8)}...)
		case uint16:
			{
				buffer = append(buffer, WORD_STACK_STUB...)
				buffer = append(buffer, byte(i*8))
				v := args[i].(uint16)
				for _, b := range *(*[2]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
			}
		case uint32:
			{
				buffer = append(buffer, DWORD_STACK_STUB...)
				buffer = append(buffer, byte(i*8))
				v := args[i].(uint32)
				for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
			}
		case uint64:
			{
				buffer = append(buffer, 0x48, 0xB8) // mov rax,QWORD
				v := args[i].(uint64)
				for _, b := range *(*[8]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
				buffer = append(buffer, QWORD_STACK_STUB...)
				buffer = append(buffer, byte(i*8))
			}
		case float32:
			{
				buffer = append(buffer, DWORD_STACK_STUB...)
				buffer = append(buffer, byte(i*8))
				v := args[i].(float32)
				for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
			}
		case float64:
			{
				buffer = append(buffer, 0x48, 0xB8) // mov rax,QWORD
				v := args[i].(float64)
				for _, b := range *(*[8]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
				buffer = append(buffer, QWORD_STACK_STUB...)
				buffer = append(buffer, byte(i*8))
			}
		}
	}

	for i := 0; i < le && i < 4; i++ {
		switch args[i].(type) {
		case uint8:
			buffer = append(buffer, BYTE_REG_STUBS[i]...)
			buffer = append(buffer, args[i].(uint8))
		case uint16:
			{
				buffer = append(buffer, WORD_REG_STUBS[i]...)
				v := args[i].(uint16)
				for _, b := range *(*[2]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
			}
		case uint32:
			{
				buffer = append(buffer, DWORD_REG_STUBS[i]...)
				v := args[i].(uint32)
				for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
			}
		case uint64:
			{
				buffer = append(buffer, QWORD_REG_STUBS[i]...)
				v := args[i].(uint64)
				for _, b := range *(*[8]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
			}
		case float32:
			{
				buffer = append(buffer, 0xB8) // mov eax,DWORD
				v := args[i].(float32)
				for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
				buffer = append(buffer, FLOAT_REG_STUBS[i]...)
			}
		case float64:
			{
				buffer = append(buffer, 0x48, 0xB8) // mov rax,QWORD
				v := args[i].(float64)
				for _, b := range *(*[8]byte)(unsafe.Pointer(&v)) {
					buffer = append(buffer, b)
				}
				buffer = append(buffer, DOUBLE_REG_STUBS[i]...)
			}
		}
	}

	buffer = append(buffer, []byte{0xFF, 0x15, 0x02, 0x00, 0x00, 0x00, 0xEB, 0x08}...)
	for _, b := range *(*[8]byte)(unsafe.Pointer(&addr)) {
		buffer = append(buffer, b)
	}
	buffer = append(buffer, []byte{0x48, 0x83, 0xC4, shadow}...)

	if ret := VirtualAllocEx(h, 0, 8, w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); ret != 0 {
		if flags&0x02 != 0 || flags&0x04 != 0 {
			buffer = append(buffer, []byte{0x66, 0x48, 0x0F, 0x7E, 0xC0}...)
		}

		buffer = append(buffer, []byte{0x48, 0xA3}...)
		for _, b := range *(*[8]byte)(unsafe.Pointer(&ret)) {
			buffer = append(buffer, b)
		}
		buffer = append(buffer, 0xC3)

		if proc := VirtualAllocEx(h, 0, uint32(len(buffer)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); proc != 0 && WriteProcessMemory(h, proc, uintptr(unsafe.Pointer(&buffer[0])), uint(len(buffer))) == nil {
			if thread := CreateRemoteThread(h, proc, 0); thread != 0 {
				if flags&0x10 == 0 {
					windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)

					if bytes, err := ReadProcessMemory(h, ret, 8); err == nil {
						for i, b := range bytes {
							raw_ret[i] = b
						}
					}
				}

				w32.CloseHandle(thread)
			}

			w32.VirtualFreeEx(h, proc, 0, w32.MEM_RELEASE)
		}

		w32.VirtualFreeEx(h, ret, 0, w32.MEM_RELEASE)
	}

	return
}

func GetProcAddressEx32(h w32.HANDLE, id uint32, module string, proc string) (r uint32) {
	if module := GetModuleInfoByName(id, module); module.ModBaseSize != 0 {
		stub := []byte{0x68, 0x00, 0x00, 0x00, 0x00, 0x68, 0x00, 0x00, 0x00, 0x00, 0xE8, 0x00, 0x00, 0x00, 0x00, 0xA3, 0x00, 0x00, 0x00, 0x00, 0xC3}
		mod_base := uint32(uintptr(unsafe.Pointer(module.ModBaseAddr)))

		kernel := GetModuleInfoByName(id, "kernel32.dll")
		if p := FindPattern(h, []byte{0x8B, 0xFF, 0x55, 0x8B, 0xEC, 0xFF, 0x75, 0x04, 0xFF, 0x75, 0x0C, 0xFF, 0x75, 0x08}, []byte("xxxxxxxxxxxxxx"), uint64(uintptr(unsafe.Pointer(kernel.ModBaseAddr))), uint64(kernel.ModBaseSize)); p != 0 {
			if base := VirtualAllocEx(h, 0, uint32(len(stub)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); base != 0 {
				*(*uint32)(unsafe.Pointer(&stub[11])) = uint32(p) - uint32(uintptr(unsafe.Pointer(base))) - 15

				arg := append([]byte(proc), 0)
				if proc := VirtualAllocEx(h, 0, uint32(len(arg)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); proc != 0 && WriteProcessMemory(h, proc, uintptr(unsafe.Pointer(&arg[0])), uint(len(arg))) == nil {
					*(*uint32)(unsafe.Pointer(&stub[1])) = uint32(uintptr(unsafe.Pointer(proc)))
					*(*uint32)(unsafe.Pointer(&stub[6])) = mod_base

					if ret := VirtualAllocEx(h, 0, 4, w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); ret != 0 {
						*(*uint32)(unsafe.Pointer(&stub[16])) = uint32(uintptr(unsafe.Pointer(ret)))
						if WriteProcessMemory(h, base, uintptr(unsafe.Pointer(&stub[0])), uint(len(stub))) == nil {
							if thread := CreateRemoteThread(h, base, 0); thread != 0 {
								windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)
								w32.CloseHandle(thread)

								if data, err := ReadProcessMemory(h, ret, 4); err == nil {
									r = *(*uint32)(unsafe.Pointer(&data[0]))
								}
							}
						}

						w32.VirtualFreeEx(h, ret, 0, w32.MEM_RELEASE)
					}

					w32.VirtualFreeEx(h, proc, 0, w32.MEM_RELEASE)
				}

				w32.VirtualFreeEx(h, base, 0, w32.MEM_RELEASE)
			}
		}
	}

	return
}

func GetProcAddressEx64(h w32.HANDLE, id uint32, module string, proc string) (r uint64) {
	dll := syscall.MustLoadDLL("kernel32.dll")
	addr, _ := windows.GetProcAddress(windows.Handle(dll.Handle), "GetProcAddress")
	dll.Release()

	if module := uint64(uintptr(unsafe.Pointer(GetModuleInfoByName(id, module).ModBaseAddr))); module != 0 {
		stub := []byte{0x48, 0x83, 0xEC, 0x28, 0x48, 0xB9, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x48, 0xBA, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0x15, 0x02, 0x00, 0x00, 0x00, 0xEB, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x48, 0xA3, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x48, 0x83, 0xC4, 0x28, 0xC3}

		*(*uint64)(unsafe.Pointer(&stub[6])) = module
		*(*uint64)(unsafe.Pointer(&stub[32])) = uint64(addr)

		arg := append([]byte(proc), 0)
		if proc := VirtualAllocEx(h, 0, uint32(len(arg)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); proc != 0 && WriteProcessMemory(h, proc, uintptr(unsafe.Pointer(&arg[0])), uint(len(arg))) == nil {
			*(*uint64)(unsafe.Pointer(&stub[16])) = uint64(uintptr(unsafe.Pointer(proc)))

			if ret := VirtualAllocEx(h, 0, 8, w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); ret != 0 {
				*(*uint64)(unsafe.Pointer(&stub[42])) = uint64(uintptr(unsafe.Pointer(ret)))

				if base := VirtualAllocEx(h, 0, uint32(len(stub)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); base != 0 && WriteProcessMemory(h, base, uintptr(unsafe.Pointer(&stub[0])), uint(len(stub))) == nil {
					if thread := CreateRemoteThread(h, base, 0); thread != 0 {
						windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)
						w32.CloseHandle(thread)

						if data, err := ReadProcessMemory(h, ret, 8); err == nil {
							r = *(*uint64)(unsafe.Pointer(&data[0]))
						}
					}

					w32.VirtualFreeEx(h, base, 0, w32.MEM_RELEASE)
				}

				w32.VirtualFreeEx(h, ret, 0, w32.MEM_RELEASE)
			}

			w32.VirtualFreeEx(h, proc, 0, w32.MEM_RELEASE)
		}
	}

	return
}

func SetJMP32(h w32.HANDLE, dest uint32, src uint32) error {
	b := []byte{0xE9, 0x00, 0x00, 0x00, 0x00}
	*(*uint32)(unsafe.Pointer(&b[1])) = dest - (src + 5)
	return WriteProcessMemory(h, uintptr(src), uintptr(unsafe.Pointer(&b[0])), uint(len(b)))
}

func SetJMP64(h w32.HANDLE, dest uint64, src uint64) error {
	b := []byte{0xFF, 0x25, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	*(*uint64)(unsafe.Pointer(&b[6])) = dest
	return WriteProcessMemory(h, uintptr(src), uintptr(unsafe.Pointer(&b[0])), uint(len(b)))
}

func process_GetProcAddress32(this *Variable, args []Variable) Variable {
	r := MakeVariable(VAR_NUMBER, float64(0))

	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		if id := GetId(this); id != 0 {
			r.Value = float64(GetProcAddressEx32(h, id, args[0].Value.(string), args[1].Value.(string)))
		}
	}

	return r
}

func process_GetProcAddress64(this *Variable, args []Variable) Variable {
	r := MakeVariable(VAR_NUMBER, float64(0))

	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		if id := GetId(this); id != 0 {
			r.Value = float64(GetProcAddressEx64(h, id, args[0].Value.(string), args[1].Value.(string)))
		}
	}

	return r
}

func ConvertCallArgs(args []Variable, out *[]interface{}) bool {
	var cargs []interface{}

	for _, arg := range args {
		if arg.Type == VAR_OBJECT {
			obj := *arg.Value.(*map[string]*Variable)
			val := GetProp(obj, "Value")
			if t := GetProp(obj, "Type"); t.Type == VAR_NUMBER {
				tv := int(t.Value.(float64))
				if (tv&0x80) != 0 && val.Type == VAR_ARRAY { // raw
					tv = tv & ^0x80

					var bytes []byte
					for _, b := range *val.Value.(*[]Variable) {
						if b.Type != VAR_NUMBER {
							return false
						}

						bytes = append(bytes, byte(b.Value.(float64)))
					}

					if tv%9 == len(bytes) {
						switch tv {
						case 1:
							cargs = append(cargs, *(*uint8)(unsafe.Pointer(&bytes[0])))
							continue
						case 2:
							cargs = append(cargs, *(*uint16)(unsafe.Pointer(&bytes[0])))
							continue
						case 4:
							cargs = append(cargs, *(*uint32)(unsafe.Pointer(&bytes[0])))
							continue
						case 8:
							cargs = append(cargs, *(*uint64)(unsafe.Pointer(&bytes[0])))
							continue
						case 13:
							cargs = append(cargs, *(*float32)(unsafe.Pointer(&bytes[0])))
							continue
						case 17:
							cargs = append(cargs, *(*float64)(unsafe.Pointer(&bytes[0])))
							continue
						}
					}
				} else if val.Type == VAR_NUMBER { // normal
					switch tv {
					case 1:
						cargs = append(cargs, uint8(val.Value.(float64)))
						continue
					case 2:
						cargs = append(cargs, uint16(val.Value.(float64)))
						continue
					case 4:
						cargs = append(cargs, uint32(val.Value.(float64)))
						continue
					case 8:
						cargs = append(cargs, uint64(val.Value.(float64)))
						continue
					case 13:
						cargs = append(cargs, float32(val.Value.(float64)))
						continue
					case 17:
						cargs = append(cargs, val.Value.(float64))
						continue
					}
				}
			}
		}

		return false
	}

	*out = cargs
	return true
}

func process_Call32(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		if l := len(args); l > 0 && args[0].Type == VAR_NUMBER {
			flags := uint32(0)
			var cargs []interface{}
			if l > 1 && args[1].Type == VAR_NUMBER {
				flags = uint32(args[1].Value.(float64))
			}

			if l < 2 || ConvertCallArgs(args[2:], &cargs) {
				ret := Call32(h, uint32(args[0].Value.(float64)), flags, cargs...)
				if flags&0x08 != 0 {
					var r []Variable
					for _, b := range ret {
						r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
					}
					return MakeVariable(VAR_ARRAY, &r)
				}

				if len(ret) > 0 {
					if flags&0x01 != 0 {
						return MakeVariable(VAR_NUMBER, float64(*(*int64)(unsafe.Pointer(&ret[0]))))
					} else if flags&0x02 != 0 {
						return MakeVariable(VAR_NUMBER, float64(*(*float32)(unsafe.Pointer(&ret[0]))))
					} else if flags&0x04 != 0 {
						return MakeVariable(VAR_NUMBER, *(*float64)(unsafe.Pointer(&ret[0])))
					}

					return MakeVariable(VAR_NUMBER, float64(*(*int32)(unsafe.Pointer(&ret[0]))))
				}
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Call64(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		if l := len(args); l > 0 && args[0].Type == VAR_NUMBER {
			flags := uint32(0)
			var cargs []interface{}
			if l > 1 && args[1].Type == VAR_NUMBER {
				flags = uint32(args[1].Value.(float64))
			}

			if l < 2 || ConvertCallArgs(args[2:], &cargs) {
				ret := Call64(h, uint64(args[0].Value.(float64)), flags, cargs...)
				if flags&0x08 != 0 {
					var r []Variable
					for _, b := range ret {
						r = append(r, MakeVariable(VAR_NUMBER, float64(b)))
					}
					return MakeVariable(VAR_ARRAY, &r)
				}

				if flags&0x02 != 0 {
					return MakeVariable(VAR_NUMBER, float64(*(*float32)(unsafe.Pointer(&ret[0]))))
				} else if flags&0x04 != 0 {
					return MakeVariable(VAR_NUMBER, *(*float64)(unsafe.Pointer(&ret[0])))
				}

				return MakeVariable(VAR_NUMBER, float64(*(*int64)(unsafe.Pointer(&ret[0]))))
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func PopArgs32(b *[]byte, argFlags []uint32, argsAddr uint32, stackOffset uint16) uint16 {
	buffer := *b
	for i, a := range argFlags {
		switch a {
		case 1: // int8
			fallthrough
		case 2: // int16
			fallthrough
		case 4: // int32
			fallthrough
		case 13: // float32
			buffer = append(buffer, []byte{0x8B, 0x45, byte(stackOffset + 8), 0xA3}...)
			v := argsAddr + uint32(i*8)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
				buffer = append(buffer, b)
			}
			stackOffset += 4
		case 8: // int64
			fallthrough
		case 17: // float64
			buffer = append(buffer, []byte{0x8B, 0x45, byte(stackOffset + 8), 0xA3}...)
			v := argsAddr + uint32(i*8)
			for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
				buffer = append(buffer, b)
			}

			buffer = append(buffer, []byte{0x8B, 0x45, byte(stackOffset + 12), 0xA3}...)
			v += 4
			for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
				buffer = append(buffer, b)
			}
			stackOffset += 8
		}
	}
	*b = buffer
	return stackOffset
}

func process_Hook32(this *Variable, args []Variable) Variable {
	obj := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if h := GetHandle(this); h != 0 {
		if id := GetId(this); id != 0 {
			if GetModuleInfoByName(id, "user32.dll").ModBaseSize == 0 {
				process_LoadLibrary32(this, []Variable{MakeVariable(VAR_STRING, `user32.dll`)})
			}

			if len(args) > 3 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER && args[2].Type == VAR_ARRAY && args[3].Type == VAR_FUNCTION {
				flags := uint32(args[1].Value.(float64))

				var argFlags []uint32
				for _, arg := range *args[2].Value.(*[]Variable) {
					if arg.Type != VAR_NUMBER {
						return MakeVariable(VAR_NUMBER, float64(0))
					}

					argFlags = append(argFlags, uint32(arg.Value.(float64)))
				}

				le := len(argFlags)
				if data := VirtualAllocEx(h, 0, 12+uint32(8*le), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); data != 0 {
					addr := uint32(args[0].Value.(float64))
					threadIdAddr := uint32(data)
					retAddr := uint32(data + 4)
					argsAddr := uint32(data + 12)
					stackPop := uint16(0)

					VirtualProtectEx(h, uintptr(addr), 16, w32.PAGE_EXECUTE_READWRITE)

					buffer := []byte{0x55}
					if le > 0 {
						buffer = append(buffer, []byte{0x8B, 0xEC}...)

						if flags&0x20 != 0 { // stdcall
							stackPop = PopArgs32(&buffer, argFlags, argsAddr, stackPop)
						} else if flags&0x40 != 0 { // fastcall - ecx(index 0 dword or less), edx(index 1 dword or less), rest stack
							state := 0
							stubs := [][]byte{{0x89, 0x0D}, {0x89, 0x15}}

							for i, a := range argFlags {
								if state < 2 && (a == 1 || a == 2 || a == 4) {
									buffer = append(buffer, stubs[i]...)
									v := argsAddr + uint32(i*8)
									for _, b := range *(*[4]byte)(unsafe.Pointer(&v)) {
										buffer = append(buffer, b)
									}
									state++
								} else {
									stackPop = PopArgs32(&buffer, []uint32{a}, argsAddr+uint32(i*8), stackPop)
								}
							}
						} else if flags&0x80 != 0 { // thiscall - ecx(this)
							buffer = append(buffer, []byte{0x89, 0x0D}...)
							for _, b := range *(*[4]byte)(unsafe.Pointer(&argsAddr)) {
								buffer = append(buffer, b)
							}

							if le > 1 {
								stackPop = PopArgs32(&buffer, argFlags[1:], argsAddr+8, stackPop)
							}
						} else { // cdecl - caller cleanup
							PopArgs32(&buffer, argFlags, argsAddr, stackPop)
						}
					}

					// Store current thread ID
					AbsCall32(&buffer, GetProcAddressEx32(h, id, "kernel32.dll", "GetCurrentThreadId"))
					buffer = append(buffer, 0xA3)
					for _, b := range *(*[4]byte)(unsafe.Pointer(&threadIdAddr)) {
						buffer = append(buffer, b)
					}

					// Call PostThreadMessageA with PID and address
					var cargs []interface{}
					cargs = append(cargs, uint32(msgThreadId))
					cargs = append(cargs, uint32(660))
					cargs = append(cargs, uint32(id))
					cargs = append(cargs, uint32(addr))
					PushArgs32(&buffer, cargs)
					AbsCall32(&buffer, GetProcAddressEx32(h, id, "user32.dll", "PostThreadMessageA"))

					// Get thread pseudo-handle in EAX
					AbsCall32(&buffer, GetProcAddressEx32(h, id, "kernel32.dll", "GetCurrentThread"))

					// Push EAX to stack and suspend thread
					buffer = append(buffer, 0x50)
					AbsCall32(&buffer, GetProcAddressEx32(h, id, "kernel32.dll", "SuspendThread"))

					// Return
					if flags&0x01 != 0 { // int64
						buffer = append(buffer, 0xA1)
						for _, b := range *(*[4]byte)(unsafe.Pointer(&retAddr)) {
							buffer = append(buffer, b)
						}

						buffer = append(buffer, []byte{0x8B, 0x15}...)
						half := retAddr + 4
						for _, b := range *(*[4]byte)(unsafe.Pointer(&half)) {
							buffer = append(buffer, b)
						}
					} else if flags&0x02 != 0 { // float32
						buffer = append(buffer, []byte{0xD9, 0x05}...)
						for _, b := range *(*[4]byte)(unsafe.Pointer(&retAddr)) {
							buffer = append(buffer, b)
						}
					} else if flags&0x04 != 0 { // float64
						buffer = append(buffer, []byte{0xDD, 0x05}...)
						for _, b := range *(*[4]byte)(unsafe.Pointer(&retAddr)) {
							buffer = append(buffer, b)
						}
					} else { // int32
						buffer = append(buffer, 0xA1)
						for _, b := range *(*[4]byte)(unsafe.Pointer(&retAddr)) {
							buffer = append(buffer, b)
						}
					}

					buffer = append(buffer, []byte{0x5D, 0xC2}...)
					for _, b := range *(*[2]byte)(unsafe.Pointer(&stackPop)) {
						buffer = append(buffer, b)
					}

					if proc := VirtualAllocEx(h, 0, uint32(len(buffer)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); proc != 0 && WriteProcessMemory(h, proc, uintptr(unsafe.Pointer(&buffer[0])), uint(len(buffer))) == nil {
						if patch, err := ReadProcessMemory(h, uintptr(addr), 5); err == nil {
							AddProp(&obj, "Address", MakeVariable(VAR_NUMBER, float64(addr)))
							AddProp(&obj, "Hook", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
								SetJMP32(h, uint32(proc), addr)
								return MakeVariable(VAR_NUMBER, float64(0))
							}))
							AddProp(&obj, "Unhook", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
								WriteProcessMemory(h, uintptr(addr), uintptr(unsafe.Pointer(&patch[0])), uint(len(patch)))
								return MakeVariable(VAR_NUMBER, float64(0))
							}))
							AddProp(&obj, "Free", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
								WriteProcessMemory(h, uintptr(addr), uintptr(unsafe.Pointer(&patch[0])), uint(len(patch)))
								w32.VirtualFreeEx(h, proc, 0, w32.MEM_RELEASE)
								w32.VirtualFreeEx(h, data, 0, w32.MEM_RELEASE)
								msgHooksMutex.Lock()
								for i, hook := range msgHooks {
									if hook.ProcessId == id && hook.Address == uint64(addr) {
										msgHooks = append(msgHooks[:i], msgHooks[i+1:]...)
										break
									}
								}
								msgHooksMutex.Unlock()
								return MakeVariable(VAR_NUMBER, float64(0))
							}))

							msgHooksMutex.Lock()
							msgHooks = append(msgHooks, MessageHook{
								ProcessHandle:   h,
								ProcessId:       id,
								Address:         uint64(addr),
								ThreadIdAddress: uint64(threadIdAddr),
								Args:            argFlags,
								ArgsAddress:     uint64(argsAddr),
								Ret:             flags,
								RetAddress:      uint64(retAddr),
								Hook:            uint64(proc),
								Object:          &obj,
								UserHandler:     &args[3],
							})
							msgHooksMutex.Unlock()
							SetJMP32(h, uint32(proc), addr)
						}
					}
				}
			}
		}
	}

	return obj
}

func process_Hook64(this *Variable, args []Variable) Variable {
	INT_REG_STUBS := [][]byte{{0x48, 0x8B, 0xC1, 0x48, 0xA3}, {0x48, 0x8B, 0xC2, 0x48, 0xA3}, {0x49, 0x8B, 0xC0, 0x48, 0xA3}, {0x49, 0x8B, 0xC1, 0x48, 0xA3}}
	FLOAT_REG_STUBS := [][]byte{{0x66, 0x48, 0x0F, 0x7E, 0xC0, 0x48, 0xA3}, {0x66, 0x48, 0x0F, 0x7E, 0xC8, 0x48, 0xA3}, {0x66, 0x48, 0x0F, 0x7E, 0xD0, 0x48, 0xA3}, {0x66, 0x48, 0x0F, 0x7E, 0xD8, 0x48, 0xA3}}

	obj := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if h := GetHandle(this); h != 0 {
		if id := GetId(this); id != 0 {
			if GetModuleInfoByName(id, "user32.dll").ModBaseSize == 0 {
				process_LoadLibrary64(this, []Variable{MakeVariable(VAR_STRING, `user32.dll`)})
			}

			if len(args) > 3 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER && args[2].Type == VAR_ARRAY && args[3].Type == VAR_FUNCTION {
				flags := uint32(args[1].Value.(float64))

				var argFlags []uint32
				for _, arg := range *args[2].Value.(*[]Variable) {
					if arg.Type != VAR_NUMBER {
						return MakeVariable(VAR_NUMBER, float64(0))
					}

					argFlags = append(argFlags, uint32(arg.Value.(float64)))
				}

				le := len(argFlags)
				if data := VirtualAllocEx(h, 0, 12+uint32(8*le), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); data != 0 {
					addr := uint64(args[0].Value.(float64))
					threadIdAddr := uint64(data)
					retAddr := uint64(data + 4)
					argsAddr := uint64(data + 12)

					VirtualProtectEx(h, uintptr(addr), 16, w32.PAGE_EXECUTE_READWRITE)

					var buffer []byte
					// Store arguments
					for i := 0; i < le && i < 4; i++ {
						if v := argFlags[i]; v == 13 || v == 17 {
							buffer = append(buffer, FLOAT_REG_STUBS[i]...)
						} else {
							buffer = append(buffer, INT_REG_STUBS[i]...)
						}

						v := argsAddr + uint64(i*8)
						for _, b := range *(*[8]byte)(unsafe.Pointer(&v)) {
							buffer = append(buffer, b)
						}
					}

					for i := 4; i < le; i++ {
						buffer = append(buffer, []byte{0x48, 0x8B, 0x44, 0x24}...)
						buffer = append(buffer, byte(8+i*8))
						buffer = append(buffer, []byte{0x48, 0xA3}...)
						v := argsAddr + uint64(i*8)
						for _, b := range *(*[8]byte)(unsafe.Pointer(&v)) {
							buffer = append(buffer, b)
						}
					}

					// Store current thread ID
					buffer = append(buffer, []byte{0x48, 0x83, 0xEC, 0x28, 0xFF, 0x15, 0x02, 0x00, 0x00, 0x00, 0xEB, 0x08}...)
					qword := uint64(GetProcAddressEx64(h, id, "kernel32.dll", "GetCurrentThreadId"))
					for _, b := range *(*[8]byte)(unsafe.Pointer(&qword)) {
						buffer = append(buffer, b)
					}
					buffer = append(buffer, 0xA3)
					for _, b := range *(*[8]byte)(unsafe.Pointer(&threadIdAddr)) {
						buffer = append(buffer, b)
					}

					// Call PostThreadMessageA with PID and address
					buffer = append(buffer, 0xB9)
					dword := uint32(msgThreadId)
					for _, b := range *(*[4]byte)(unsafe.Pointer(&dword)) {
						buffer = append(buffer, b)
					}

					buffer = append(buffer, 0xBA)
					dword = 660
					for _, b := range *(*[4]byte)(unsafe.Pointer(&dword)) {
						buffer = append(buffer, b)
					}

					buffer = append(buffer, []byte{0x49, 0xB8}...)
					qword = uint64(id)
					for _, b := range *(*[8]byte)(unsafe.Pointer(&qword)) {
						buffer = append(buffer, b)
					}

					buffer = append(buffer, []byte{0x49, 0xB9}...)
					for _, b := range *(*[8]byte)(unsafe.Pointer(&addr)) {
						buffer = append(buffer, b)
					}

					buffer = append(buffer, []byte{0xFF, 0x15, 0x02, 0x00, 0x00, 0x00, 0xEB, 0x08}...)
					qword = GetProcAddressEx64(h, id, "user32.dll", "PostThreadMessageA")
					for _, b := range *(*[8]byte)(unsafe.Pointer(&qword)) {
						buffer = append(buffer, b)
					}

					// Store thread psuedo-handle in RCX
					buffer = append(buffer, []byte{0xFF, 0x15, 0x02, 0x00, 0x00, 0x00, 0xEB, 0x08}...)
					qword = uint64(GetProcAddressEx64(h, id, "kernel32.dll", "GetCurrentThread"))
					for _, b := range *(*[8]byte)(unsafe.Pointer(&qword)) {
						buffer = append(buffer, b)
					}
					buffer = append(buffer, []byte{0x48, 0x8B, 0xC8}...)

					// Suspend thread
					buffer = append(buffer, []byte{0xFF, 0x15, 0x02, 0x00, 0x00, 0x00, 0xEB, 0x08}...)
					qword = uint64(GetProcAddressEx64(h, id, "kernel32.dll", "SuspendThread"))
					for _, b := range *(*[8]byte)(unsafe.Pointer(&qword)) {
						buffer = append(buffer, b)
					}
					buffer = append(buffer, []byte{0x48, 0x83, 0xC4, 0x28}...)

					// Return
					buffer = append(buffer, []byte{0x48, 0xA1}...)
					for _, b := range *(*[8]byte)(unsafe.Pointer(&retAddr)) {
						buffer = append(buffer, b)
					}
					if flags&0x02 != 0 {
						buffer = append(buffer, []byte{0x66, 0x0F, 0x6E, 0xC0}...)
					} else if flags&0x04 != 0 {
						buffer = append(buffer, []byte{0x66, 0x48, 0x0F, 0x6E, 0xC0}...)
					}
					buffer = append(buffer, 0xC3)

					if proc := VirtualAllocEx(h, 0, uint32(len(buffer)), w32.MEM_COMMIT|w32.MEM_RESERVE, w32.PAGE_EXECUTE_READWRITE); proc != 0 && WriteProcessMemory(h, proc, uintptr(unsafe.Pointer(&buffer[0])), uint(len(buffer))) == nil {
						if patch, err := ReadProcessMemory(h, uintptr(addr), 14); err == nil {
							AddProp(&obj, "Address", MakeVariable(VAR_NUMBER, float64(addr)))
							AddProp(&obj, "Hook", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
								SetJMP64(h, uint64(proc), addr)
								return MakeVariable(VAR_NUMBER, float64(0))
							}))
							AddProp(&obj, "Unhook", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
								WriteProcessMemory(h, uintptr(addr), uintptr(unsafe.Pointer(&patch[0])), uint(len(patch)))
								return MakeVariable(VAR_NUMBER, float64(0))
							}))
							AddProp(&obj, "Free", MakeVariable(VAR_NFUNCTION, func(this *Variable, args []Variable) Variable {
								WriteProcessMemory(h, uintptr(addr), uintptr(unsafe.Pointer(&patch[0])), uint(len(patch)))
								w32.VirtualFreeEx(h, proc, 0, w32.MEM_RELEASE)
								w32.VirtualFreeEx(h, data, 0, w32.MEM_RELEASE)
								msgHooksMutex.Lock()
								for i, hook := range msgHooks {
									if hook.ProcessId == id && hook.Address == addr {
										msgHooks = append(msgHooks[:i], msgHooks[i+1:]...)
										break
									}
								}
								msgHooksMutex.Unlock()
								return MakeVariable(VAR_NUMBER, float64(0))
							}))

							msgHooksMutex.Lock()
							msgHooks = append(msgHooks, MessageHook{
								ProcessHandle:   h,
								ProcessId:       id,
								Address:         addr,
								ThreadIdAddress: threadIdAddr,
								Args:            argFlags,
								ArgsAddress:     argsAddr,
								Ret:             flags,
								RetAddress:      retAddr,
								Hook:            uint64(proc),
								Object:          &obj,
								UserHandler:     &args[3],
							})
							msgHooksMutex.Unlock()
							SetJMP64(h, uint64(proc), addr)
						}
					}
				}
			}
		}
	}

	return obj
}

func process_Suspend(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		NtSuspendProcess(h)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Resume(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		NtResumeProcess(h)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Modules(this *Variable, args []Variable) Variable {
	var modules []Variable

	if id := GetId(this); id != 0 {
		if len(args) > 0 && args[0].Type == VAR_STRING {
			entry := GetModuleInfoByName(id, args[0].Value.(string))
			m := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
			AddProp(&m, "Size", MakeVariable(VAR_NUMBER, float64(entry.ModBaseSize)))
			AddProp(&m, "Base", MakeVariable(VAR_NUMBER, float64(uintptr(unsafe.Pointer(entry.ModBaseAddr)))))
			AddProp(&m, "Name", MakeVariable(VAR_STRING, windows.UTF16ToString(entry.SzModule[:])))
			return m
		} else {
			snapshot := w32.CreateToolhelp32Snapshot(w32.TH32CS_SNAPMODULE|w32.TH32CS_SNAPMODULE32, id)
			if snapshot != 0 {
				var entry w32.MODULEENTRY32
				entry.Size = uint32(unsafe.Sizeof(entry))

				if w32.Module32First(snapshot, &entry) {
					c := true
					for c {
						m := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
						AddProp(&m, "Size", MakeVariable(VAR_NUMBER, float64(entry.ModBaseSize)))
						AddProp(&m, "Base", MakeVariable(VAR_NUMBER, float64(uintptr(unsafe.Pointer(entry.ModBaseAddr)))))
						AddProp(&m, "Name", MakeVariable(VAR_STRING, windows.UTF16ToString(entry.SzModule[:])))
						modules = append(modules, m)
						c = w32.Module32Next(snapshot, &entry)
					}
				}

				w32.CloseHandle(snapshot)
			}
		}
	}

	return MakeVariable(VAR_ARRAY, &modules)
}

func process_Windows(this *Variable, args []Variable) Variable {
	var r []Variable

	if id := GetId(this); id != 0 {
		EnumWindows(syscall.NewCallback(func(hwnd w32.HWND, param uintptr) uintptr {
			if _, i := w32.GetWindowThreadProcessId(hwnd); uint32(i) == id {
				r = append(r, GetWindowObject(hwnd))
			}

			return 1
		}), 0)
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func GetThreadStackTop32(process w32.HANDLE, thread w32.HANDLE) uint64 {
	if SuspendThread(thread) {
		defer ResumeThread(thread)

		var context WOW64_CONTEXT
		context.ContextFlags = 0x00010000 | 0x00000004

		if Wow64GetThreadContext(thread, &context) {
			var entry WOW64_LDT_ENTRY
			if Wow64GetThreadSelectorEntry(thread, context.SegFs, &entry) {
				stack := uint32(entry.BaseLow) + (uint32(entry.BaseMid) << 16) + (uint32(entry.BaseHi) << 24) + 4

				if data, err := ReadProcessMemory(process, uintptr(stack), 4); err == nil {
					return uint64(*(*uint32)(unsafe.Pointer(&data[0]))) - 4
				}
			}
		}
	}

	return 0
}

func GetThreadStackTop64(process w32.HANDLE, thread w32.HANDLE) uint64 {
	if SuspendThread(thread) {
		defer ResumeThread(thread)

		var tbi THREAD_BASIC_INFORMATION
		NtQueryInformationThread(thread, 0, &tbi)
		if addr := uint64(uintptr(unsafe.Pointer(tbi.TebBaseAddress))); addr != 0 {
			if data, err := ReadProcessMemory(process, uintptr(uint64(addr)+8), 8); err == nil {
				return (*(*uint64)(unsafe.Pointer(&data[0]))) - 8
			}
		}
	}

	return 0
}

func GetThreadStack32(process w32.HANDLE, pid w32.DWORD, thread w32.HANDLE) uint64 {
	kernel32 := GetModuleInfoByName(uint32(pid), "kernel32.dll")
	min := uint64(uintptr(unsafe.Pointer(kernel32.ModBaseAddr)))
	max := min + uint64(kernel32.ModBaseSize)

	if top := GetThreadStackTop32(process, thread); top != 0 {
		for {
			if data, err := ReadProcessMemory(process, uintptr(top), 4); err == nil {
				if stack := uint64(*(*uint32)(unsafe.Pointer(&data[0]))); stack >= min && stack < max {
					return top
				}
			} else {
				break
			}

			top -= 4
		}
	}

	return 0
}

func GetThreadStack64(process w32.HANDLE, pid w32.DWORD, thread w32.HANDLE) uint64 {
	kernel32 := GetModuleInfoByName(uint32(pid), "kernel32.dll")
	min := uint64(uintptr(unsafe.Pointer(kernel32.ModBaseAddr)))
	max := min + uint64(kernel32.ModBaseSize)

	if top := GetThreadStackTop64(process, thread); top != 0 {
		for {
			if data, err := ReadProcessMemory(process, uintptr(top), 8); err == nil {
				if stack := (*(*uint64)(unsafe.Pointer(&data[0]))); stack >= min && stack < max {
					return top
				}
			} else {
				break
			}

			top -= 8
		}
	}

	return 0
}

func process_Threads(this *Variable, args []Variable) Variable {
	var threads []Variable

	if process := GetHandle(this); process != 0 {
		if id := w32.DWORD(GetId(this)); id != 0 {
			if v, ok := (*(*this).Value.(*map[string]*Variable))["Wow64"]; ok && v.Type == VAR_NUMBER {
				if i := int(v.Value.(float64)); i == 0 || i == 1 {
					stack_func := ([](func(w32.HANDLE, w32.DWORD, w32.HANDLE) uint64){GetThreadStack64, GetThreadStack32})[i]

					snapshot := w32.CreateToolhelp32Snapshot(w32.TH32CS_SNAPTHREAD, 0)
					if snapshot != 0 {
						var entry THREADENTRY32
						entry.DwSize = w32.DWORD(unsafe.Sizeof(entry))

						if Thread32First(snapshot, &entry) {
							thread := CreateThread(syscall.NewCallback(func() uintptr {
								c := true
								for c {
									if entry.Th32OwnerProcessID == id {
										t := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
										AddProp(&t, "Id", MakeVariable(VAR_NUMBER, float64(entry.Th32ThreadID)))
										AddProp(&t, "Owner", MakeVariable(VAR_NUMBER, float64(entry.Th32OwnerProcessID)))
										AddProp(&t, "Priority", MakeVariable(VAR_NUMBER, float64(entry.TpBasePri)))

										time := int64(0)
										stack := uint64(0)
										if handle := OpenThread(0x0008|0x0002|0x0040, w32.BOOL(0), entry.Th32ThreadID); handle != 0 {
											time = GetThreadCreationTime(handle)
											stack = stack_func(process, id, handle)
											w32.CloseHandle(handle)
										}
										AddProp(&t, "CreationTime", MakeVariable(VAR_NUMBER, float64(time)))
										AddProp(&t, "Stack", MakeVariable(VAR_NUMBER, float64(stack)))
										AddProp(&t, "Suspend", MakeVariable(VAR_NFUNCTION, thread_Suspend))
										AddProp(&t, "Resume", MakeVariable(VAR_NFUNCTION, thread_Resume))

										threads = append(threads, t)
									}
									c = Thread32Next(snapshot, &entry)
								}

								return 0
							}), 0)

							w32.WaitForSingleObject(thread, w32.INFINITE)
							w32.CloseHandle(thread)
						}
					}
				}
			}
		}
	}

	sort.SliceStable(threads, func(i, j int) bool {
		return (*(*threads[i].Value.(*map[string]*Variable))["CreationTime"]).Value.(float64) < (*(*threads[j].Value.(*map[string]*Variable))["CreationTime"]).Value.(float64)
	})

	return MakeVariable(VAR_ARRAY, &threads)
}

func process_LoadLibrary32(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_STRING {
		if id := GetId(this); id != 0 {
			path := windows.StringToUTF16(args[0].Value.(string))
			le := uint32(len(path) + 1)
			if arg := VirtualAllocEx(h, 0, le, w32.MEM_RESERVE|w32.MEM_COMMIT, w32.PAGE_EXECUTE_READWRITE); arg != 0 && WriteProcessMemory(h, arg, uintptr(unsafe.Pointer(&path[0])), uint(le)) == nil {
				if thread := CreateRemoteThread(h, uintptr(GetProcAddressEx32(h, id, "kernel32.dll", "LoadLibraryW")), arg); thread != 0 {
					windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)
					w32.CloseHandle(thread)
				}

				w32.VirtualFreeEx(h, arg, 0, w32.MEM_RELEASE)
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_LoadLibrary64(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_STRING {
		path := windows.StringToUTF16(args[0].Value.(string))
		le := uint32(len(path) + 1)

		dll := syscall.MustLoadDLL("kernel32.dll")
		addr, _ := windows.GetProcAddress(windows.Handle(dll.Handle), "LoadLibraryW")
		dll.Release()

		if arg := VirtualAllocEx(h, 0, le, w32.MEM_RESERVE|w32.MEM_COMMIT, w32.PAGE_EXECUTE_READWRITE); arg != 0 && WriteProcessMemory(h, arg, uintptr(unsafe.Pointer(&path[0])), uint(le)) == nil {
			if thread := CreateRemoteThread(h, addr, arg); thread != 0 {
				windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)
				w32.CloseHandle(thread)
			}

			w32.VirtualFreeEx(h, arg, 0, w32.MEM_RELEASE)
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func thread_Suspend(this *Variable, args []Variable) Variable {
	if id := w32.DWORD(GetId(this)); id != 0 {
		if handle := OpenThread(0x0002, w32.BOOL(0), id); handle != 0 {
			SuspendThread(handle)
			w32.CloseHandle(handle)
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func thread_Resume(this *Variable, args []Variable) Variable {
	if id := w32.DWORD(GetId(this)); id != 0 {
		if handle := OpenThread(0x0002, w32.BOOL(0), id); handle != 0 {
			ResumeThread(handle)
			w32.CloseHandle(handle)
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Close(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		w32.CloseHandle(h)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Exit(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		code := uint(0)
		if len(args) > 0 && args[0].Type == VAR_NUMBER {
			code = uint(args[0].Value.(float64))
		}
		w32.TerminateProcess(h, code)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func MaskCompare(a []byte, b []byte, mask []byte) bool {
	for i := 0; i < len(mask); i++ {
		if mask[i] == byte('x') && a[i] != b[i] {
			return false
		}
	}

	return true
}

func FindPattern(handle w32.HANDLE, pattern []byte, mask []byte, min uint64, size uint64) uint64 {
	max := uint64(0x7FFFFFFFFFFF)
	if size != 0 {
		max = min + size
	}

	var mi MEMORY_BASIC_INFORMATION
	base := uintptr(min)
	for base < uintptr(max) && VirtualQueryEx(handle, w32.LPCVOID(base), uintptr(unsafe.Pointer(&mi)), w32.SIZE_T(unsafe.Sizeof(mi))) != 0 {
		if mi.State&(w32.MEM_COMMIT|w32.MEM_RESERVE) != 0 && mi.Protect != 0 {
			if data, err := ReadProcessMemory(handle, uintptr(mi.BaseAddress), uint(mi.RegionSize)); err == nil {
				for i := 0; i < len(data)-len(mask); i++ {
					if MaskCompare(data[i:], pattern, mask) {
						return uint64(uintptr(mi.BaseAddress)) + uint64(i)
					}
				}
			}
		}
		base += uintptr(mi.RegionSize)
	}

	return 0
}

func process_FindPattern(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		if l := len(args); l == 1 {
			if args[0].Type == VAR_STRING {
				if v := strings.Replace(args[0].Value.(string), " ", "", -1); len(v)%2 == 0 {
					var pattern []byte
					var mask []byte

					for i := 0; i < len(v); i += 2 {
						if v[i] == '?' {
							pattern = append(pattern, 0)
							mask = append(mask, '?')
						} else {
							b, err := strconv.ParseInt(v[i:i+2], 16, 64)
							if err != nil {
								return MakeVariable(VAR_NUMBER, float64(0))
							}

							pattern = append(pattern, byte(b))
							mask = append(mask, 'x')
						}
					}

					return MakeVariable(VAR_NUMBER, float64(FindPattern(h, pattern, mask, 0, 0)))
				}
			} else if args[0].Type == VAR_ARRAY {
				var b []byte
				var mask []byte
				for _, e := range *args[0].Value.(*[]Variable) {
					if e.Type != VAR_NUMBER {
						return MakeVariable(VAR_NUMBER, float64(0))
					}
					b = append(b, byte(e.Value.(float64)))
					mask = append(mask, byte('x'))
				}
				return MakeVariable(VAR_NUMBER, float64(FindPattern(h, b, mask, 0, 0)))
			}
		} else if l > 1 && args[1].Type == VAR_STRING {
			mask := []byte(args[1].Value.(string))
			if args[0].Type == VAR_ARRAY {
				var b []byte
				for _, e := range *args[0].Value.(*[]Variable) {
					if e.Type != VAR_NUMBER {
						return MakeVariable(VAR_NUMBER, float64(0))
					}
					b = append(b, byte(e.Value.(float64)))
				}
				return MakeVariable(VAR_NUMBER, float64(FindPattern(h, b, mask, 0, 0)))
			} else if args[0].Type == VAR_STRING {
				return MakeVariable(VAR_NUMBER, float64(FindPattern(h, []byte(args[0].Value.(string)), mask, 0, 0)))
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadPointer32(this *Variable, args []Variable) Variable {
	var base uint32

	if h := GetHandle(this); h != 0 {
		i := 0
		for ; i < len(args)-1; i++ {
			if args[i].Type != VAR_NUMBER {
				break
			}

			data, err := ReadProcessMemory(h, uintptr(base+uint32(args[i].Value.(float64))), uint(unsafe.Sizeof(base)))
			if err != nil {
				break
			}

			base = binary.LittleEndian.Uint32(data)
		}

		if args[i].Type == VAR_NUMBER {
			base += uint32(args[i].Value.(float64))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(base))
}

func process_ReadPointer64(this *Variable, args []Variable) Variable {
	var base uint64

	if h := GetHandle(this); h != 0 {
		i := 0
		for ; i < len(args)-1; i++ {
			if args[i].Type != VAR_NUMBER {
				break
			}

			data, err := ReadProcessMemory(h, uintptr(base+uint64(args[i].Value.(float64))), uint(unsafe.Sizeof(base)))
			if err != nil {
				break
			}

			base = binary.LittleEndian.Uint64(data)
		}

		if args[i].Type == VAR_NUMBER {
			base += uint64(args[i].Value.(float64))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(base))
}

func process_Protect(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 2 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER && args[2].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, float64(VirtualProtectEx(h, uintptr(uint64(args[0].Value.(float64))), uint32(args[1].Value.(float64)), uint32(args[2].Value.(float64)))))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Alloc(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		if l := len(args); l > 0 && args[0].Type == VAR_NUMBER {
			size := uint32(args[0].Value.(float64))
			alloc_type := uint32(w32.MEM_COMMIT | w32.MEM_RESERVE)
			protect := uint32(w32.PAGE_EXECUTE_READWRITE)

			if l > 1 && args[1].Type == VAR_NUMBER {
				alloc_type = uint32(args[1].Value.(float64))
				if l > 2 && args[1].Type == VAR_NUMBER {
					protect = uint32(args[2].Value.(float64))
				}
			}

			return MakeVariable(VAR_NUMBER, float64(VirtualAllocEx(h, uintptr(0), size, alloc_type, protect)))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Free(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		if l := len(args); l > 0 && args[0].Type == VAR_NUMBER {
			addr := uintptr(uint64(args[0].Value.(float64)))
			size := uintptr(0)
			free_type := uint32(w32.MEM_RELEASE)

			if l > 2 && args[1].Type == VAR_NUMBER && args[2].Type == VAR_NUMBER {
				size = uintptr(uint32(args[1].Value.(float64)))
				free_type = uint32(args[2].Value.(float64))
			}

			if w32.VirtualFreeEx(h, addr, size, free_type) {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Read(this *Variable, args []Variable) Variable {
	var bytes []Variable

	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), uint(args[1].Value.(float64)))
		if err == nil {
			for _, b := range data {
				bytes = append(bytes, MakeVariable(VAR_NUMBER, float64(b)))
			}
		}
	}

	return MakeVariable(VAR_ARRAY, &bytes)
}

func process_Write(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_ARRAY {
		var b []byte

		for _, e := range *args[1].Value.(*[]Variable) {
			if e.Type != VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, float64(0))
			}

			b = append(b, byte(e.Value.(float64)))
		}

		if WriteProcessMemory(h, uintptr(uint64(args[0].Value.(float64))), uintptr(unsafe.Pointer(&b[0])), uint(len(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadString(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		var str []byte

		base := uint64(args[0].Value.(float64))
	loop:
		for {
			bytes, _ := ReadProcessMemory(h, uintptr(base), 0x1000)
			if len(bytes) == 0 {
				break
			}

			for _, b := range bytes {
				if b == 0 {
					break loop
				}

				str = append(str, b)
			}

			base += 0x1000
		}

		return MakeVariable(VAR_STRING, string(str))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteString(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_STRING {
		b := append([]byte(args[1].Value.(string)), 0)
		if WriteProcessMemory(h, uintptr(uint64(args[0].Value.(float64))), uintptr(unsafe.Pointer(&b[0])), uint(len(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadString16(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		var str []uint16

		base := uint64(args[0].Value.(float64))
	loop:
		for {
			bytes, _ := ReadProcessMemory(h, uintptr(base), 0x1000)
			if l := len(bytes); l == 0 {
				break
			} else if l%2 != 0 {
				bytes = bytes[0 : l-1]
			}

			for i := 0; i < len(bytes); i += 2 {
				v := *(*uint16)(unsafe.Pointer(&bytes[i]))
				if v == 0 {
					break loop
				}

				str = append(str, v)
			}

			base += 0x1000
		}

		return MakeVariable(VAR_STRING, windows.UTF16ToString(str))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteString16(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_STRING {
		if b, err := windows.UTF16FromString(args[1].Value.(string)); err == nil {
			if WriteProcessMemory(h, uintptr(uint64(args[0].Value.(float64))), uintptr(unsafe.Pointer(&b[0])), uint(len(b)*2)) == nil {
				return MakeVariable(VAR_NUMBER, float64(1))
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadInt16(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 2)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*int16)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteInt16(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := int16(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadInt32(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*int32)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteInt32(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := int32(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadInt64(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 8)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*int64)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteInt64(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := int64(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadUint16(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 2)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*uint16)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteUint16(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := uint16(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadUint32(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*uint32)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteUint32(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := uint32(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadUint64(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 8)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*uint64)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteUint64(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := uint64(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadFloat(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*float32)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteFloat(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := float32(args[1].Value.(float64))
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadDouble(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, *(*float64)(unsafe.Pointer(&data[0])))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteDouble(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := args[1].Value.(float64)
		if WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b))) == nil {
			return MakeVariable(VAR_NUMBER, float64(1))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/***********************************************/
/*                     window                  */
/***********************************************/
func GetWindowObject(hwnd w32.HWND) Variable {
	window := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if hwnd != 0 {
		AddProp(&window, "Handle", MakeVariable(VAR_NUMBER, float64(hwnd)))

		AddProp(&window, "SendMessage", MakeVariable(VAR_NFUNCTION, window_SendMessage))
		AddProp(&window, "Title", MakeVariable(VAR_NFUNCTION, window_Title))
		AddProp(&window, "Position", MakeVariable(VAR_NFUNCTION, window_Position))
		AddProp(&window, "Size", MakeVariable(VAR_NFUNCTION, window_Size))
		AddProp(&window, "Focus", MakeVariable(VAR_NFUNCTION, window_Focus))
		AddProp(&window, "Minimize", MakeVariable(VAR_NFUNCTION, window_Minimize))
		AddProp(&window, "Maximize", MakeVariable(VAR_NFUNCTION, window_Maximize))
		AddProp(&window, "Hide", MakeVariable(VAR_NFUNCTION, window_Hide))
		AddProp(&window, "Show", MakeVariable(VAR_NFUNCTION, window_Show))
		AddProp(&window, "Class", MakeVariable(VAR_STRING, w32.GetClassNameW(hwnd)))

		tid, pid := w32.GetWindowThreadProcessId(hwnd)
		AddProp(&window, "ThreadId", MakeVariable(VAR_NUMBER, float64(tid)))
		AddProp(&window, "ProcessId", MakeVariable(VAR_NUMBER, float64(pid)))

		AddProp(&window, "Parent", GetWindowObject(GetParent(hwnd)))
		AddProp(&window, "Children", MakeVariable(VAR_NFUNCTION, window_Children))
	}

	return window
}

func window_Foreground(this *Variable, args []Variable) Variable {
	hwnd, _ := w32.GetForegroundWindow()
	return GetWindowObject(w32.HWND(hwnd))
}

func window_List(this *Variable, args []Variable) Variable {
	var r []Variable

	w32.EnumChildWindows(0, func(hwnd w32.HWND, lparam w32.LPARAM) w32.LRESULT {
		r = append(r, GetWindowObject(hwnd))
		return 1
	}, 0)

	return MakeVariable(VAR_ARRAY, &r)
}

func window_SendMessage(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 && len(args) > 2 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER && args[2].Type == VAR_NUMBER {
		return MakeVariable(VAR_NUMBER, float64(w32.SendMessage(w32.HWND(h), uint32(args[0].Value.(float64)), uintptr(uint64(args[1].Value.(float64))), uintptr(uint64(args[2].Value.(float64))))))
	}
	return MakeVariable(VAR_NUMBER, float64(0))
}

func window_Title(this *Variable, args []Variable) Variable {
	s := ""
	if h := GetHandle(this); h != 0 {
		if len(args) > 0 && args[0].Type == VAR_STRING {
			w32.SetWindowText(w32.HWND(h), args[0].Value.(string))
			return MakeVariable(VAR_NUMBER, float64(0))
		} else {
			title := make([]uint16, 0xFFF)
			w32.GetWindowTextW(syscall.Handle(h), &title[0], 0xFFF)
			s = windows.UTF16ToString(title)
		}
	}
	return MakeVariable(VAR_STRING, s)
}

func window_Position(this *Variable, args []Variable) Variable {
	pos := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		w32.SetWindowPos(w32.HWND(h), 0, int(args[0].Value.(float64)), int(args[1].Value.(float64)), 0, 0, w32.SWP_NOZORDER|w32.SWP_NOSIZE)
		return MakeVariable(VAR_NUMBER, float64(0))
	} else {
		rect := w32.GetWindowRect(w32.HWND(h))
		AddProp(&pos, "X", MakeVariable(VAR_NUMBER, float64(rect.Left)))
		AddProp(&pos, "Y", MakeVariable(VAR_NUMBER, float64(rect.Top)))
	}

	return pos
}

func window_Size(this *Variable, args []Variable) Variable {
	size := MakeVariable(VAR_OBJECT, &map[string]*Variable{})

	if h := GetHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		w32.ShowWindow(w32.HWND(h), w32.SW_RESTORE)
		w32.SetWindowPos(w32.HWND(h), 0, 0, 0, int(args[0].Value.(float64)), int(args[1].Value.(float64)), w32.SWP_NOZORDER|w32.SWP_NOMOVE)
		return MakeVariable(VAR_NUMBER, float64(0))
	} else {
		rect := w32.GetWindowRect(w32.HWND(h))
		AddProp(&size, "Width", MakeVariable(VAR_NUMBER, float64(rect.Right-rect.Left)))
		AddProp(&size, "Height", MakeVariable(VAR_NUMBER, float64(rect.Bottom-rect.Top)))
	}

	return size
}

func window_Focus(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		w32.SetForegroundWindow(w32.HWND(h))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func window_Minimize(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		w32.ShowWindow(w32.HWND(h), w32.SW_MINIMIZE)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func window_Maximize(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		w32.ShowWindow(w32.HWND(h), w32.SW_MAXIMIZE)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func window_Hide(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		w32.ShowWindow(w32.HWND(h), w32.SW_HIDE)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func window_Show(this *Variable, args []Variable) Variable {
	if h := GetHandle(this); h != 0 {
		w32.ShowWindow(w32.HWND(h), w32.SW_SHOW)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func window_Children(this *Variable, args []Variable) Variable {
	var r []Variable

	if h := GetHandle(this); h != 0 {
		w32.EnumChildWindows(w32.HWND(h), func(hwnd w32.HWND, lparam w32.LPARAM) w32.LRESULT {
			r = append(r, GetWindowObject(hwnd))
			return 1
		}, 0)
	}

	return MakeVariable(VAR_ARRAY, &r)
}

/***********************************************/
/*                      input                  */
/***********************************************/
var input_OnKeyDownFunc *Variable
var input_OnKeyUpFunc *Variable

func KeyboardHook(code int, wParam w32.WPARAM, lParam w32.LPARAM) w32.LRESULT {
	if code >= 0 {
		keycode := (*(*w32.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))).VkCode

		if wParam == w32.WM_KEYDOWN && input_OnKeyDownFunc != nil {
			CallUserFunc(input_OnKeyDownFunc, nil, []Variable{MakeVariable(VAR_NUMBER, float64(keycode))})
		} else if wParam == w32.WM_KEYUP && input_OnKeyUpFunc != nil {
			CallUserFunc(input_OnKeyUpFunc, nil, []Variable{MakeVariable(VAR_NUMBER, float64(keycode))})
		}
	}

	return w32.CallNextHookEx(0, code, wParam, lParam)
}

func input_OnKeyDown(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_FUNCTION {
		input_OnKeyDownFunc = &args[0]
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func input_OnKeyUp(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_FUNCTION {
		input_OnKeyUpFunc = &args[0]
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func input_KeyDown(this *Variable, args []Variable) Variable {
	var inputs []w32.INPUT

	for _, e := range args {
		if e.Type != VAR_NUMBER {
			break
		}

		k := uint(e.Value.(float64))
		i := w32.INPUT{Type: w32.INPUT_KEYBOARD}
		i.Ki.WScan = MapVirtualKey(k, w32.MAPVK_VK_TO_VSC)
		i.Ki.WVk = uint16(k)
		inputs = append(inputs, i)
	}

	if len(inputs) > 0 {
		w32.SendInput(inputs)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func input_KeyUp(this *Variable, args []Variable) Variable {
	var inputs []w32.INPUT

	for _, e := range args {
		if e.Type != VAR_NUMBER {
			break
		}

		k := uint(e.Value.(float64))
		i := w32.INPUT{Type: w32.INPUT_KEYBOARD}
		i.Ki.WScan = MapVirtualKey(k, w32.MAPVK_VK_TO_VSC)
		i.Ki.DwFlags = 2
		i.Ki.WVk = uint16(k)
		inputs = append(inputs, i)
	}

	if len(inputs) > 0 {
		w32.SendInput(inputs)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func input_IsKeyDown(this *Variable, args []Variable) Variable {
	for _, e := range args {
		if e.Type != VAR_NUMBER || w32.GetAsyncKeyState(int(e.Value.(float64)))&(1<<15) == 0 {
			return MakeVariable(VAR_NUMBER, float64(0))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(1))
}

func input_SendKeys(this *Variable, args []Variable) Variable {
	if len(args) > 0 && args[0].Type == VAR_STRING {
		var inputs []w32.INPUT

		s := args[0].Value.(string)
		for i := 0; i < len(s); i++ {
			inp := w32.INPUT{Type: w32.INPUT_KEYBOARD}
			inp.Ki.WScan = MapVirtualKey(uint(s[i]), w32.MAPVK_VK_TO_VSC)
			inp.Ki.WVk = uint16(VkKeyScanA(s[i]))
			inputs = append(inputs, inp)
			inp.Ki.DwFlags = 2
			inputs = append(inputs, inp)
		}

		if len(inputs) > 0 {
			w32.SendInput(inputs)
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

/* helper functions */
func CallUserFunc(v *Variable, this *Variable, args []Variable) Variable {
	f := v.Value.([]Tree)

	var thread []Stack
	stack := StackAdd(&thread, -1)

	StackPush("arguments", &thread, stack, MakeVariable(VAR_ARRAY, &args))
	if this != nil {
		StackPush("this", &thread, stack, *this)
	}

	for i, e := range f[0].C {
		if i < len(args) {
			StackPush(e.T.Value.(string), &thread, stack, args[i])
		} else {
			StackPush(e.T.Value.(string), &thread, stack, MakeVariable(VAR_NUMBER, float64(0)))
		}
	}

	ret := MakeVariable(VAR_NUMBER, float64(0))

	for _, e := range f[1].C {
		if v := Eval(e, &thread, stack); v.Type == VAR_RETURN {
			ret = v.Value.(Variable)
			break
		}
	}

	StackRemove(&thread)
	return ret
}

func GetProp(m map[string]*Variable, name string) Variable {
	if v, ok := m[name]; ok {
		return *v
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func AddProp(o *Variable, name string, value Variable) {
	value.Parent = o
	(*(*o).Value.(*map[string]*Variable))[name] = &value
}

func InitStack() {
	StackPush("true", nil, -1, MakeVariable(VAR_NUMBER, float64(1)))
	StackPush("false", nil, -1, MakeVariable(VAR_NUMBER, float64(0)))
	StackPush("len", nil, -1, MakeVariable(VAR_NFUNCTION, global_len))
	StackPush("type", nil, -1, MakeVariable(VAR_NFUNCTION, global_type))
	StackPush("copy", nil, -1, MakeVariable(VAR_NFUNCTION, global_copy))
	var arguments []Variable
	for _, s := range os.Args {
		arguments = append(arguments, MakeVariable(VAR_STRING, s))
	}
	StackPush("arguments", nil, -1, MakeVariable(VAR_ARRAY, &arguments))

	object := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&object, "Keys", MakeVariable(VAR_NFUNCTION, object_Keys))
	StackPush("object", nil, -1, object)

	array := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&array, "Each", MakeVariable(VAR_NFUNCTION, array_Each))
	AddProp(&array, "Find", MakeVariable(VAR_NFUNCTION, array_Find))
	AddProp(&array, "Insert", MakeVariable(VAR_NFUNCTION, array_Insert))
	AddProp(&array, "Pop", MakeVariable(VAR_NFUNCTION, array_Pop))
	AddProp(&array, "Push", MakeVariable(VAR_NFUNCTION, array_Push))
	AddProp(&array, "Remove", MakeVariable(VAR_NFUNCTION, array_Remove))
	AddProp(&array, "Sort", MakeVariable(VAR_NFUNCTION, array_Sort))
	StackPush("array", nil, -1, array)

	console := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&console, "Print", MakeVariable(VAR_NFUNCTION, console_Print))
	AddProp(&console, "Println", MakeVariable(VAR_NFUNCTION, console_Println))
	AddProp(&console, "ReadLine", MakeVariable(VAR_NFUNCTION, console_ReadLine))
	AddProp(&console, "Clear", MakeVariable(VAR_NFUNCTION, console_Clear))
	StackPush("console", nil, -1, console)

	number := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&number, "ToString", MakeVariable(VAR_NFUNCTION, number_ToString))
	AddProp(&number, "FromString", MakeVariable(VAR_NFUNCTION, string_ToNumber))
	AddProp(&number, "ToInt16Bytes", MakeVariable(VAR_NFUNCTION, number_ToInt16Bytes))
	AddProp(&number, "ToUint16Bytes", MakeVariable(VAR_NFUNCTION, number_ToUint16Bytes))
	AddProp(&number, "ToIntBytes", MakeVariable(VAR_NFUNCTION, number_ToInt32Bytes))
	AddProp(&number, "ToInt32Bytes", MakeVariable(VAR_NFUNCTION, number_ToInt32Bytes))
	AddProp(&number, "ToUintBytes", MakeVariable(VAR_NFUNCTION, number_ToUint32Bytes))
	AddProp(&number, "ToUint32Bytes", MakeVariable(VAR_NFUNCTION, number_ToUint32Bytes))
	AddProp(&number, "ToInt64Bytes", MakeVariable(VAR_NFUNCTION, number_ToInt64Bytes))
	AddProp(&number, "ToUint64Bytes", MakeVariable(VAR_NFUNCTION, number_ToUint64Bytes))
	AddProp(&number, "ToFloat32Bytes", MakeVariable(VAR_NFUNCTION, number_ToFloat32Bytes))
	AddProp(&number, "ToFloatBytes", MakeVariable(VAR_NFUNCTION, number_ToFloat32Bytes))
	AddProp(&number, "ToFloat64Bytes", MakeVariable(VAR_NFUNCTION, number_ToFloat64Bytes))
	AddProp(&number, "ToDoubleBytes", MakeVariable(VAR_NFUNCTION, number_ToFloat64Bytes))
	AddProp(&number, "FromInt16Bytes", MakeVariable(VAR_NFUNCTION, number_FromInt16Bytes))
	AddProp(&number, "FromUint16Bytes", MakeVariable(VAR_NFUNCTION, number_FromUint16Bytes))
	AddProp(&number, "FromIntBytes", MakeVariable(VAR_NFUNCTION, number_FromInt32Bytes))
	AddProp(&number, "FromUintBytes", MakeVariable(VAR_NFUNCTION, number_FromUint32Bytes))
	AddProp(&number, "FromInt32Bytes", MakeVariable(VAR_NFUNCTION, number_FromInt32Bytes))
	AddProp(&number, "FromUint32Bytes", MakeVariable(VAR_NFUNCTION, number_FromUint32Bytes))
	AddProp(&number, "FromInt64Bytes", MakeVariable(VAR_NFUNCTION, number_FromInt64Bytes))
	AddProp(&number, "FromUint64Bytes", MakeVariable(VAR_NFUNCTION, number_FromUint64Bytes))
	AddProp(&number, "FromFloat32Bytes", MakeVariable(VAR_NFUNCTION, number_FromFloat32Bytes))
	AddProp(&number, "FromFloatBytes", MakeVariable(VAR_NFUNCTION, number_FromFloat32Bytes))
	AddProp(&number, "FromFloat64Bytes", MakeVariable(VAR_NFUNCTION, number_FromFloat64Bytes))
	AddProp(&number, "FromDoubleBytes", MakeVariable(VAR_NFUNCTION, number_FromFloat64Bytes))
	StackPush("number", nil, -1, number)

	str := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&str, "CharCodeAt", MakeVariable(VAR_NFUNCTION, string_CharCodeAt))
	AddProp(&str, "Contains", MakeVariable(VAR_NFUNCTION, string_Contains))
	AddProp(&str, "FromBytes", MakeVariable(VAR_NFUNCTION, string_FromBytes))
	AddProp(&str, "FromCharCode", MakeVariable(VAR_NFUNCTION, string_FromCharCode))
	AddProp(&str, "FromNumber", MakeVariable(VAR_NFUNCTION, number_ToString))
	AddProp(&str, "IndexOf", MakeVariable(VAR_NFUNCTION, string_IndexOf))
	AddProp(&str, "LastIndexOf", MakeVariable(VAR_NFUNCTION, string_LastIndexOf))
	AddProp(&str, "Replace", MakeVariable(VAR_NFUNCTION, string_Replace))
	AddProp(&str, "Slice", MakeVariable(VAR_NFUNCTION, string_Slice))
	AddProp(&str, "Split", MakeVariable(VAR_NFUNCTION, string_Split))
	AddProp(&str, "ToBytes", MakeVariable(VAR_NFUNCTION, string_ToBytes))
	AddProp(&str, "ToLower", MakeVariable(VAR_NFUNCTION, string_ToLower))
	AddProp(&str, "ToNumber", MakeVariable(VAR_NFUNCTION, string_ToNumber))
	AddProp(&str, "ToUpper", MakeVariable(VAR_NFUNCTION, string_ToUpper))
	AddProp(&str, "Trim", MakeVariable(VAR_NFUNCTION, string_Trim))
	StackPush("string", nil, -1, str)

	regex := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&regex, "Find", MakeVariable(VAR_NFUNCTION, regex_Find))
	AddProp(&regex, "FindIndex", MakeVariable(VAR_NFUNCTION, regex_FindIndex))
	AddProp(&regex, "Match", MakeVariable(VAR_NFUNCTION, regex_Match))
	AddProp(&regex, "Replace", MakeVariable(VAR_NFUNCTION, regex_Replace))
	AddProp(&regex, "Split", MakeVariable(VAR_NFUNCTION, regex_Split))
	StackPush("regex", nil, -1, regex)

	math_ := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&math_, "DEG_RAD", MakeVariable(VAR_NUMBER, 180/math.Pi))
	AddProp(&math_, "E", MakeVariable(VAR_NUMBER, math.E))
	AddProp(&math_, "LN10", MakeVariable(VAR_NUMBER, math.Ln10))
	AddProp(&math_, "LN2", MakeVariable(VAR_NUMBER, math.Ln2))
	AddProp(&math_, "LOG10E", MakeVariable(VAR_NUMBER, math.Log10E))
	AddProp(&math_, "LOG2E", MakeVariable(VAR_NUMBER, math.Log2E))
	AddProp(&math_, "PHI", MakeVariable(VAR_NUMBER, math.Phi))
	AddProp(&math_, "PI", MakeVariable(VAR_NUMBER, math.Pi))
	AddProp(&math_, "RAD_DEG", MakeVariable(VAR_NUMBER, math.Pi/180))
	AddProp(&math_, "SQRT1_2", MakeVariable(VAR_NUMBER, math.Sqrt(1.0/2.0)))
	AddProp(&math_, "SQRT2", MakeVariable(VAR_NUMBER, math.Sqrt2))
	AddProp(&math_, "Abs", MakeVariable(VAR_NFUNCTION, math_Abs))
	AddProp(&math_, "Acos", MakeVariable(VAR_NFUNCTION, math_Acos))
	AddProp(&math_, "Acosh", MakeVariable(VAR_NFUNCTION, math_Acosh))
	AddProp(&math_, "Asin", MakeVariable(VAR_NFUNCTION, math_Asinh))
	AddProp(&math_, "Atan", MakeVariable(VAR_NFUNCTION, math_Atan))
	AddProp(&math_, "Atan2", MakeVariable(VAR_NFUNCTION, math_Atan2))
	AddProp(&math_, "Cbrt", MakeVariable(VAR_NFUNCTION, math_Cbrt))
	AddProp(&math_, "Ceil", MakeVariable(VAR_NFUNCTION, math_Ceil))
	AddProp(&math_, "Cos", MakeVariable(VAR_NFUNCTION, math_Cos))
	AddProp(&math_, "Cosh", MakeVariable(VAR_NFUNCTION, math_Cosh))
	AddProp(&math_, "Exp", MakeVariable(VAR_NFUNCTION, math_Exp))
	AddProp(&math_, "Expm1", MakeVariable(VAR_NFUNCTION, math_Expm1))
	AddProp(&math_, "Floor", MakeVariable(VAR_NFUNCTION, math_Floor))
	AddProp(&math_, "Hypot", MakeVariable(VAR_NFUNCTION, math_Hypot))
	AddProp(&math_, "Log", MakeVariable(VAR_NFUNCTION, math_Log))
	AddProp(&math_, "Log10", MakeVariable(VAR_NFUNCTION, math_Log10))
	AddProp(&math_, "Log1p", MakeVariable(VAR_NFUNCTION, math_Log1p))
	AddProp(&math_, "Log2", MakeVariable(VAR_NFUNCTION, math_Log2))
	AddProp(&math_, "Max", MakeVariable(VAR_NFUNCTION, math_Max))
	AddProp(&math_, "Min", MakeVariable(VAR_NFUNCTION, math_Min))
	AddProp(&math_, "Pow", MakeVariable(VAR_NFUNCTION, math_Pow))
	AddProp(&math_, "Random", MakeVariable(VAR_NFUNCTION, math_Random))
	AddProp(&math_, "Round", MakeVariable(VAR_NFUNCTION, math_Round))
	AddProp(&math_, "Sign", MakeVariable(VAR_NFUNCTION, math_Sign))
	AddProp(&math_, "Sin", MakeVariable(VAR_NFUNCTION, math_Sin))
	AddProp(&math_, "Sinh", MakeVariable(VAR_NFUNCTION, math_Sinh))
	AddProp(&math_, "Sqrt", MakeVariable(VAR_NFUNCTION, math_Sqrt))
	AddProp(&math_, "Tan", MakeVariable(VAR_NFUNCTION, math_Tan))
	AddProp(&math_, "Tanh", MakeVariable(VAR_NFUNCTION, math_Tanh))
	AddProp(&math_, "Trunc", MakeVariable(VAR_NFUNCTION, math_Trunc))
	StackPush("math", nil, -1, math_)

	json_ := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&json_, "Stringify", MakeVariable(VAR_NFUNCTION, json_Stringify))
	AddProp(&json_, "Parse", MakeVariable(VAR_NFUNCTION, json_Parse))
	StackPush("json", nil, -1, json_)

	http_ := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&http_, "Get", MakeVariable(VAR_NFUNCTION, http_Get))
	AddProp(&http_, "Request", MakeVariable(VAR_NFUNCTION, http_Request))
	StackPush("http", nil, -1, http_)

	thread := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&thread, "Sleep", MakeVariable(VAR_NFUNCTION, thread_Sleep))
	AddProp(&thread, "Create", MakeVariable(VAR_NFUNCTION, thread_Create))
	StackPush("thread", nil, -1, thread)

	dll := syscall.MustLoadDLL("kernel32.dll")
	procGetCurrentProcessId, _ = dll.FindProc("GetCurrentProcessId")
	procCreateThread, _ = dll.FindProc("CreateThread")
	procCreateRemoteThread, _ = dll.FindProc("CreateRemoteThread")
	procIsWow64Process, _ = dll.FindProc("IsWow64Process")
	procReadProcessMemory, _ = dll.FindProc("ReadProcessMemory")
	procWriteProcessMemory, _ = dll.FindProc("WriteProcessMemory")
	procVirtualQueryEx, _ = dll.FindProc("VirtualQueryEx")
	procOpenThread, _ = dll.FindProc("OpenThread")
	procSuspendThread, _ = dll.FindProc("SuspendThread")
	procResumeThread, _ = dll.FindProc("ResumeThread")
	procThread32First, _ = dll.FindProc("Thread32First")
	procThread32Next, _ = dll.FindProc("Thread32Next")
	procWow64GetThreadContext, _ = dll.FindProc("Wow64GetThreadContext")
	procWow64GetThreadSelectorEntry, _ = dll.FindProc("Wow64GetThreadSelectorEntry")
	procGetThreadTimes, _ = dll.FindProc("GetThreadTimes")
	procVirtualAllocEx, _ = dll.FindProc("VirtualAllocEx")
	procVirtualProtectEx, _ = dll.FindProc("VirtualProtectEx")
	procQueryFullProcessImageNameW, _ = dll.FindProc("QueryFullProcessImageNameW")
	dll.Release()

	dll = syscall.MustLoadDLL("user32.dll")
	procMapVirtualKey, _ = dll.FindProc("MapVirtualKeyW")
	procVkKeyScanA, _ = dll.FindProc("VkKeyScanA")
	procEnumWindows, _ = dll.FindProc("EnumWindows")
	procGetParent, _ = dll.FindProc("GetParent")
	dll.Release()

	dll = syscall.MustLoadDLL("winmm.dll")
	procTimeGetTime, _ = dll.FindProc("timeGetTime")
	dll.Release()

	dll = syscall.MustLoadDLL("ntdll.dll")
	procNtSuspendProcess, _ = dll.FindProc("NtSuspendProcess")
	procNtResumeProcess, _ = dll.FindProc("NtResumeProcess")
	procNtQueryInformationThread, _ = dll.FindProc("NtQueryInformationThread")
	dll.Release()

	date := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&date, "Now", MakeVariable(VAR_NFUNCTION, date_Now))
	AddProp(&date, "Time", MakeVariable(VAR_NFUNCTION, date_Time))
	StackPush("date", nil, -1, date)

	file := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&file, "SEEK_SET", MakeVariable(VAR_NUMBER, float64(os.SEEK_SET)))
	AddProp(&file, "SEEK_CUR", MakeVariable(VAR_NUMBER, float64(os.SEEK_CUR)))
	AddProp(&file, "SEEK_END", MakeVariable(VAR_NUMBER, float64(os.SEEK_END)))
	AddProp(&file, "Stdout", FileObject(os.Stdout))
	AddProp(&file, "Stdin", FileObject(os.Stdin))
	AddProp(&file, "Stderr", FileObject(os.Stderr))
	AddProp(&file, "Open", MakeVariable(VAR_NFUNCTION, file_Open))
	AddProp(&file, "Remove", MakeVariable(VAR_NFUNCTION, file_Remove))
	AddProp(&file, "RemoveAll", MakeVariable(VAR_NFUNCTION, file_RemoveAll))
	StackPush("file", nil, -1, file)

	process := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&process, "Open", MakeVariable(VAR_NFUNCTION, process_Open))
	AddProp(&process, "List", MakeVariable(VAR_NFUNCTION, process_List))
	AddProp(&process, "Current", process_Open(nil, []Variable{MakeVariable(VAR_NUMBER, float64(GetCurrentProcessId()))}))

	AddProp(&process, "PAGE_EXECUTE", MakeVariable(VAR_NUMBER, float64(0x10)))
	AddProp(&process, "PAGE_EXECUTE_READ", MakeVariable(VAR_NUMBER, float64(0x20)))
	AddProp(&process, "PAGE_EXECUTE_READWRITE", MakeVariable(VAR_NUMBER, float64(0x40)))
	AddProp(&process, "PAGE_EXECUTE_WRITECOPY", MakeVariable(VAR_NUMBER, float64(0x80)))
	AddProp(&process, "PAGE_NOACCESS", MakeVariable(VAR_NUMBER, float64(0x01)))
	AddProp(&process, "PAGE_READONLY", MakeVariable(VAR_NUMBER, float64(0x02)))
	AddProp(&process, "PAGE_READWRITE", MakeVariable(VAR_NUMBER, float64(0x04)))
	AddProp(&process, "PAGE_WRITECOPY", MakeVariable(VAR_NUMBER, float64(0x08)))
	AddProp(&process, "PAGE_TARGETS_INVALID", MakeVariable(VAR_NUMBER, float64(0x40000000)))
	AddProp(&process, "PAGE_TARGETS_NO_UPDATE", MakeVariable(VAR_NUMBER, float64(0x40000000)))
	AddProp(&process, "PAGE_GUARD", MakeVariable(VAR_NUMBER, float64(0x100)))
	AddProp(&process, "PAGE_NOCACHE", MakeVariable(VAR_NUMBER, float64(0x200)))
	AddProp(&process, "PAGE_WRITECOMBINE", MakeVariable(VAR_NUMBER, float64(0x400)))

	AddProp(&process, "MEM_COMMIT", MakeVariable(VAR_NUMBER, float64(0x00001000)))
	AddProp(&process, "MEM_RESERVE", MakeVariable(VAR_NUMBER, float64(0x00002000)))
	AddProp(&process, "MEM_RESET", MakeVariable(VAR_NUMBER, float64(0x00080000)))
	AddProp(&process, "MEM_RESET_UNDO", MakeVariable(VAR_NUMBER, float64(0x1000000)))
	AddProp(&process, "MEM_LARGE_PAGES", MakeVariable(VAR_NUMBER, float64(0x20000000)))
	AddProp(&process, "MEM_PHYSICAL", MakeVariable(VAR_NUMBER, float64(0x00400000)))
	AddProp(&process, "MEM_TOP_DOWN", MakeVariable(VAR_NUMBER, float64(0x00100000)))
	AddProp(&process, "MEM_WRITE_WATCH", MakeVariable(VAR_NUMBER, float64(0x00200000)))
	AddProp(&process, "MEM_COALESCE_PLACEHOLDERS", MakeVariable(VAR_NUMBER, float64(0x00000001)))
	AddProp(&process, "MEM_PRESERVE_PLACEHOLDER", MakeVariable(VAR_NUMBER, float64(0x00000002)))
	AddProp(&process, "MEM_DECOMMIT", MakeVariable(VAR_NUMBER, float64(0x4000)))
	AddProp(&process, "MEM_RELEASE", MakeVariable(VAR_NUMBER, float64(0x8000)))

	AddProp(&process, "FUNC_RET_INT", MakeVariable(VAR_NUMBER, float64(0x00)))
	AddProp(&process, "FUNC_RET_INT32", MakeVariable(VAR_NUMBER, float64(0x00)))
	AddProp(&process, "FUNC_RET_INT64", MakeVariable(VAR_NUMBER, float64(0x01)))
	AddProp(&process, "FUNC_RET_FLOAT", MakeVariable(VAR_NUMBER, float64(0x02)))
	AddProp(&process, "FUNC_RET_FLOAT32", MakeVariable(VAR_NUMBER, float64(0x02)))
	AddProp(&process, "FUNC_RET_FLOAT64", MakeVariable(VAR_NUMBER, float64(0x04)))
	AddProp(&process, "FUNC_RET_RAW", MakeVariable(VAR_NUMBER, float64(0x08)))
	AddProp(&process, "FUNC_RET_NONE", MakeVariable(VAR_NUMBER, float64(0x10)))
	AddProp(&process, "FUNC_CDECL", MakeVariable(VAR_NUMBER, float64(0x00)))
	AddProp(&process, "FUNC_STDCALL", MakeVariable(VAR_NUMBER, float64(0x20)))
	AddProp(&process, "FUNC_FASTCALL", MakeVariable(VAR_NUMBER, float64(0x40)))
	AddProp(&process, "FUNC_THISCALL", MakeVariable(VAR_NUMBER, float64(0x80)))

	AddProp(&process, "ARG_INT8", MakeVariable(VAR_NUMBER, float64(1)))
	AddProp(&process, "ARG_INT16", MakeVariable(VAR_NUMBER, float64(2)))
	AddProp(&process, "ARG_INT", MakeVariable(VAR_NUMBER, float64(4)))
	AddProp(&process, "ARG_INT32", MakeVariable(VAR_NUMBER, float64(4)))
	AddProp(&process, "ARG_INT64", MakeVariable(VAR_NUMBER, float64(8)))
	AddProp(&process, "ARG_FLOAT", MakeVariable(VAR_NUMBER, float64(13)))
	AddProp(&process, "ARG_FLOAT32", MakeVariable(VAR_NUMBER, float64(13)))
	AddProp(&process, "ARG_FLOAT64", MakeVariable(VAR_NUMBER, float64(17)))
	AddProp(&process, "ARG_RAW", MakeVariable(VAR_NUMBER, float64(0x80)))
	StackPush("process", nil, -1, process)

	window := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&window, "Foreground", MakeVariable(VAR_NFUNCTION, window_Foreground))
	AddProp(&window, "List", MakeVariable(VAR_NFUNCTION, window_List))
	StackPush("window", nil, -1, window)

	input := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&input, "OnKeyDown", MakeVariable(VAR_NFUNCTION, input_OnKeyDown))
	AddProp(&input, "OnKeyUp", MakeVariable(VAR_NFUNCTION, input_OnKeyUp))
	AddProp(&input, "KeyDown", MakeVariable(VAR_NFUNCTION, input_KeyDown))
	AddProp(&input, "KeyUp", MakeVariable(VAR_NFUNCTION, input_KeyUp))
	AddProp(&input, "IsKeyDown", MakeVariable(VAR_NFUNCTION, input_IsKeyDown))
	AddProp(&input, "SendKeys", MakeVariable(VAR_NFUNCTION, input_SendKeys))
	key := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&key, "LBUTTON", MakeVariable(VAR_NUMBER, float64(0x01)))
	AddProp(&key, "RBUTTON", MakeVariable(VAR_NUMBER, float64(0x02)))
	AddProp(&key, "CANCEL", MakeVariable(VAR_NUMBER, float64(0x03)))
	AddProp(&key, "MBUTTON", MakeVariable(VAR_NUMBER, float64(0x04)))
	AddProp(&key, "XBUTTON1", MakeVariable(VAR_NUMBER, float64(0x05)))
	AddProp(&key, "XBUTTON2", MakeVariable(VAR_NUMBER, float64(0x06)))
	AddProp(&key, "BACK", MakeVariable(VAR_NUMBER, float64(0x08)))
	AddProp(&key, "BACKSPACE", MakeVariable(VAR_NUMBER, float64(0x08)))
	AddProp(&key, "TAB", MakeVariable(VAR_NUMBER, float64(0x09)))
	AddProp(&key, "CLEAR", MakeVariable(VAR_NUMBER, float64(0x0C)))
	AddProp(&key, "RETURN", MakeVariable(VAR_NUMBER, float64(0x0D)))
	AddProp(&key, "SHIFT", MakeVariable(VAR_NUMBER, float64(0x10)))
	AddProp(&key, "CONTROL", MakeVariable(VAR_NUMBER, float64(0x11)))
	AddProp(&key, "MENU", MakeVariable(VAR_NUMBER, float64(0x12)))
	AddProp(&key, "PAUSE", MakeVariable(VAR_NUMBER, float64(0x13)))
	AddProp(&key, "CAPITAL", MakeVariable(VAR_NUMBER, float64(0x14)))
	AddProp(&key, "CAPSLOCK", MakeVariable(VAR_NUMBER, float64(0x14)))
	AddProp(&key, "CAPS", MakeVariable(VAR_NUMBER, float64(0x14)))
	AddProp(&key, "KANA", MakeVariable(VAR_NUMBER, float64(0x15)))
	AddProp(&key, "HANGUEL", MakeVariable(VAR_NUMBER, float64(0x15)))
	AddProp(&key, "HANGUL", MakeVariable(VAR_NUMBER, float64(0x15)))
	AddProp(&key, "JUNJA", MakeVariable(VAR_NUMBER, float64(0x17)))
	AddProp(&key, "FINAL", MakeVariable(VAR_NUMBER, float64(0x18)))
	AddProp(&key, "HANJA", MakeVariable(VAR_NUMBER, float64(0x19)))
	AddProp(&key, "KANJI", MakeVariable(VAR_NUMBER, float64(0x19)))
	AddProp(&key, "ESCAPE", MakeVariable(VAR_NUMBER, float64(0x1B)))
	AddProp(&key, "CONVERT", MakeVariable(VAR_NUMBER, float64(0x1C)))
	AddProp(&key, "NONCONVERT", MakeVariable(VAR_NUMBER, float64(0x1D)))
	AddProp(&key, "ACCEPT", MakeVariable(VAR_NUMBER, float64(0x1E)))
	AddProp(&key, "MODECHANGE", MakeVariable(VAR_NUMBER, float64(0x1F)))
	AddProp(&key, "SPACE", MakeVariable(VAR_NUMBER, float64(0x20)))
	AddProp(&key, "PRIOR", MakeVariable(VAR_NUMBER, float64(0x21)))
	AddProp(&key, "PAGEUP", MakeVariable(VAR_NUMBER, float64(0x21)))
	AddProp(&key, "NEXT", MakeVariable(VAR_NUMBER, float64(0x22)))
	AddProp(&key, "PAGEDOWN", MakeVariable(VAR_NUMBER, float64(0x22)))
	AddProp(&key, "END", MakeVariable(VAR_NUMBER, float64(0x23)))
	AddProp(&key, "HOME", MakeVariable(VAR_NUMBER, float64(0x24)))
	AddProp(&key, "LEFT", MakeVariable(VAR_NUMBER, float64(0x25)))
	AddProp(&key, "UP", MakeVariable(VAR_NUMBER, float64(0x26)))
	AddProp(&key, "RIGHT", MakeVariable(VAR_NUMBER, float64(0x27)))
	AddProp(&key, "DOWN", MakeVariable(VAR_NUMBER, float64(0x28)))
	AddProp(&key, "SELECT", MakeVariable(VAR_NUMBER, float64(0x29)))
	AddProp(&key, "PRINT", MakeVariable(VAR_NUMBER, float64(0x2A)))
	AddProp(&key, "EXECUTE", MakeVariable(VAR_NUMBER, float64(0x2B)))
	AddProp(&key, "SNAPSHOT", MakeVariable(VAR_NUMBER, float64(0x2C)))
	AddProp(&key, "INSERT", MakeVariable(VAR_NUMBER, float64(0x2D)))
	AddProp(&key, "DELETE", MakeVariable(VAR_NUMBER, float64(0x2E)))
	AddProp(&key, "HELP", MakeVariable(VAR_NUMBER, float64(0x2F)))
	AddProp(&key, "ZERO", MakeVariable(VAR_NUMBER, float64(0x30)))
	AddProp(&key, "_0", MakeVariable(VAR_NUMBER, float64(0x30)))
	AddProp(&key, "0", MakeVariable(VAR_NUMBER, float64(0x30)))
	AddProp(&key, "ONE", MakeVariable(VAR_NUMBER, float64(0x31)))
	AddProp(&key, "_1", MakeVariable(VAR_NUMBER, float64(0x31)))
	AddProp(&key, "1", MakeVariable(VAR_NUMBER, float64(0x31)))
	AddProp(&key, "TWO", MakeVariable(VAR_NUMBER, float64(0x32)))
	AddProp(&key, "_2", MakeVariable(VAR_NUMBER, float64(0x32)))
	AddProp(&key, "2", MakeVariable(VAR_NUMBER, float64(0x32)))
	AddProp(&key, "THREE", MakeVariable(VAR_NUMBER, float64(0x33)))
	AddProp(&key, "_3", MakeVariable(VAR_NUMBER, float64(0x33)))
	AddProp(&key, "3", MakeVariable(VAR_NUMBER, float64(0x33)))
	AddProp(&key, "FOUR", MakeVariable(VAR_NUMBER, float64(0x34)))
	AddProp(&key, "_4", MakeVariable(VAR_NUMBER, float64(0x34)))
	AddProp(&key, "4", MakeVariable(VAR_NUMBER, float64(0x34)))
	AddProp(&key, "FIVE", MakeVariable(VAR_NUMBER, float64(0x35)))
	AddProp(&key, "_5", MakeVariable(VAR_NUMBER, float64(0x35)))
	AddProp(&key, "5", MakeVariable(VAR_NUMBER, float64(0x35)))
	AddProp(&key, "SIX", MakeVariable(VAR_NUMBER, float64(0x36)))
	AddProp(&key, "_6", MakeVariable(VAR_NUMBER, float64(0x36)))
	AddProp(&key, "6", MakeVariable(VAR_NUMBER, float64(0x36)))
	AddProp(&key, "SEVEN", MakeVariable(VAR_NUMBER, float64(0x37)))
	AddProp(&key, "_7", MakeVariable(VAR_NUMBER, float64(0x37)))
	AddProp(&key, "7", MakeVariable(VAR_NUMBER, float64(0x37)))
	AddProp(&key, "EIGHT", MakeVariable(VAR_NUMBER, float64(0x38)))
	AddProp(&key, "_8", MakeVariable(VAR_NUMBER, float64(0x38)))
	AddProp(&key, "8", MakeVariable(VAR_NUMBER, float64(0x38)))
	AddProp(&key, "NINE", MakeVariable(VAR_NUMBER, float64(0x39)))
	AddProp(&key, "_9", MakeVariable(VAR_NUMBER, float64(0x39)))
	AddProp(&key, "9", MakeVariable(VAR_NUMBER, float64(0x39)))
	AddProp(&key, "A", MakeVariable(VAR_NUMBER, float64(0x41)))
	AddProp(&key, "B", MakeVariable(VAR_NUMBER, float64(0x42)))
	AddProp(&key, "C", MakeVariable(VAR_NUMBER, float64(0x43)))
	AddProp(&key, "D", MakeVariable(VAR_NUMBER, float64(0x44)))
	AddProp(&key, "E", MakeVariable(VAR_NUMBER, float64(0x45)))
	AddProp(&key, "F", MakeVariable(VAR_NUMBER, float64(0x46)))
	AddProp(&key, "G", MakeVariable(VAR_NUMBER, float64(0x47)))
	AddProp(&key, "H", MakeVariable(VAR_NUMBER, float64(0x48)))
	AddProp(&key, "I", MakeVariable(VAR_NUMBER, float64(0x49)))
	AddProp(&key, "J", MakeVariable(VAR_NUMBER, float64(0x4A)))
	AddProp(&key, "K", MakeVariable(VAR_NUMBER, float64(0x4B)))
	AddProp(&key, "L", MakeVariable(VAR_NUMBER, float64(0x4C)))
	AddProp(&key, "M", MakeVariable(VAR_NUMBER, float64(0x4D)))
	AddProp(&key, "N", MakeVariable(VAR_NUMBER, float64(0x4E)))
	AddProp(&key, "O", MakeVariable(VAR_NUMBER, float64(0x4F)))
	AddProp(&key, "P", MakeVariable(VAR_NUMBER, float64(0x50)))
	AddProp(&key, "Q", MakeVariable(VAR_NUMBER, float64(0x51)))
	AddProp(&key, "R", MakeVariable(VAR_NUMBER, float64(0x52)))
	AddProp(&key, "S", MakeVariable(VAR_NUMBER, float64(0x53)))
	AddProp(&key, "T", MakeVariable(VAR_NUMBER, float64(0x54)))
	AddProp(&key, "U", MakeVariable(VAR_NUMBER, float64(0x55)))
	AddProp(&key, "V", MakeVariable(VAR_NUMBER, float64(0x56)))
	AddProp(&key, "W", MakeVariable(VAR_NUMBER, float64(0x57)))
	AddProp(&key, "X", MakeVariable(VAR_NUMBER, float64(0x58)))
	AddProp(&key, "Y", MakeVariable(VAR_NUMBER, float64(0x59)))
	AddProp(&key, "Z", MakeVariable(VAR_NUMBER, float64(0x5A)))
	AddProp(&key, "LWIN", MakeVariable(VAR_NUMBER, float64(0x5B)))
	AddProp(&key, "RWIN", MakeVariable(VAR_NUMBER, float64(0x5C)))
	AddProp(&key, "APPS", MakeVariable(VAR_NUMBER, float64(0x5D)))
	AddProp(&key, "SLEEP", MakeVariable(VAR_NUMBER, float64(0x5F)))
	AddProp(&key, "NUMPAD0", MakeVariable(VAR_NUMBER, float64(0x60)))
	AddProp(&key, "NUMPAD1", MakeVariable(VAR_NUMBER, float64(0x61)))
	AddProp(&key, "NUMPAD2", MakeVariable(VAR_NUMBER, float64(0x62)))
	AddProp(&key, "NUMPAD3", MakeVariable(VAR_NUMBER, float64(0x63)))
	AddProp(&key, "NUMPAD4", MakeVariable(VAR_NUMBER, float64(0x64)))
	AddProp(&key, "NUMPAD5", MakeVariable(VAR_NUMBER, float64(0x65)))
	AddProp(&key, "NUMPAD6", MakeVariable(VAR_NUMBER, float64(0x66)))
	AddProp(&key, "NUMPAD7", MakeVariable(VAR_NUMBER, float64(0x67)))
	AddProp(&key, "NUMPAD8", MakeVariable(VAR_NUMBER, float64(0x68)))
	AddProp(&key, "NUMPAD9", MakeVariable(VAR_NUMBER, float64(0x69)))
	AddProp(&key, "MULTIPLY", MakeVariable(VAR_NUMBER, float64(0x6A)))
	AddProp(&key, "ADD", MakeVariable(VAR_NUMBER, float64(0x6B)))
	AddProp(&key, "SEPARATOR", MakeVariable(VAR_NUMBER, float64(0x6C)))
	AddProp(&key, "SUBTRACT", MakeVariable(VAR_NUMBER, float64(0x6D)))
	AddProp(&key, "DECIMAL", MakeVariable(VAR_NUMBER, float64(0x6E)))
	AddProp(&key, "DIVIDE", MakeVariable(VAR_NUMBER, float64(0x6F)))
	AddProp(&key, "F1", MakeVariable(VAR_NUMBER, float64(0x70)))
	AddProp(&key, "F2", MakeVariable(VAR_NUMBER, float64(0x71)))
	AddProp(&key, "F3", MakeVariable(VAR_NUMBER, float64(0x72)))
	AddProp(&key, "F4", MakeVariable(VAR_NUMBER, float64(0x73)))
	AddProp(&key, "F5", MakeVariable(VAR_NUMBER, float64(0x74)))
	AddProp(&key, "F6", MakeVariable(VAR_NUMBER, float64(0x75)))
	AddProp(&key, "F7", MakeVariable(VAR_NUMBER, float64(0x76)))
	AddProp(&key, "F8", MakeVariable(VAR_NUMBER, float64(0x77)))
	AddProp(&key, "F9", MakeVariable(VAR_NUMBER, float64(0x78)))
	AddProp(&key, "F10", MakeVariable(VAR_NUMBER, float64(0x79)))
	AddProp(&key, "F11", MakeVariable(VAR_NUMBER, float64(0x7A)))
	AddProp(&key, "F12", MakeVariable(VAR_NUMBER, float64(0x7B)))
	AddProp(&key, "F13", MakeVariable(VAR_NUMBER, float64(0x7C)))
	AddProp(&key, "F14", MakeVariable(VAR_NUMBER, float64(0x7D)))
	AddProp(&key, "F15", MakeVariable(VAR_NUMBER, float64(0x7E)))
	AddProp(&key, "F16", MakeVariable(VAR_NUMBER, float64(0x7F)))
	AddProp(&key, "F17", MakeVariable(VAR_NUMBER, float64(0x80)))
	AddProp(&key, "F18", MakeVariable(VAR_NUMBER, float64(0x81)))
	AddProp(&key, "F19", MakeVariable(VAR_NUMBER, float64(0x82)))
	AddProp(&key, "F20", MakeVariable(VAR_NUMBER, float64(0x83)))
	AddProp(&key, "F21", MakeVariable(VAR_NUMBER, float64(0x84)))
	AddProp(&key, "F22", MakeVariable(VAR_NUMBER, float64(0x85)))
	AddProp(&key, "F23", MakeVariable(VAR_NUMBER, float64(0x86)))
	AddProp(&key, "F24", MakeVariable(VAR_NUMBER, float64(0x87)))
	AddProp(&input, "Key", key)
	StackPush("input", nil, -1, input)

	msgHookCallback := syscall.NewCallback(func(ptr uintptr) uintptr {
		hook := *(*MessageHook)(unsafe.Pointer(ptr))
		if threadData, err := ReadProcessMemory(hook.ProcessHandle, uintptr(hook.ThreadIdAddress), 4); err == nil {
			if thread := OpenThread(0x2, 0, *(*w32.DWORD)(unsafe.Pointer(&threadData[0]))); thread != 0 {
				var args []Variable
				if len(hook.Args) > 0 {
					if argData, err := ReadProcessMemory(hook.ProcessHandle, uintptr(hook.ArgsAddress), uint(8*len(hook.Args))); err == nil {
						for i, a := range hook.Args {
							if (a & 0x80) != 0 {
								var r []Variable
								for b := 0; b < int(a%9); b++ {
									r = append(r, MakeVariable(VAR_NUMBER, float64(argData[(i*8)+b])))
								}
								args = append(args, MakeVariable(VAR_ARRAY, &r))
							} else {
								switch a {
								case 1:
									args = append(args, MakeVariable(VAR_NUMBER, float64(*(*int8)(unsafe.Pointer(&argData[i*8])))))
								case 2:
									args = append(args, MakeVariable(VAR_NUMBER, float64(*(*int16)(unsafe.Pointer(&argData[i*8])))))
								case 4:
									args = append(args, MakeVariable(VAR_NUMBER, float64(*(*int32)(unsafe.Pointer(&argData[i*8])))))
								case 8:
									args = append(args, MakeVariable(VAR_NUMBER, float64(*(*int64)(unsafe.Pointer(&argData[i*8])))))
								case 13:
									args = append(args, MakeVariable(VAR_NUMBER, float64(*(*float32)(unsafe.Pointer(&argData[i*8])))))
								case 17:
									args = append(args, MakeVariable(VAR_NUMBER, *(*float64)(unsafe.Pointer(&argData[i*8]))))
								}
							}
						}
					}
				}

				if ret := CallUserFunc(hook.UserHandler, hook.Object, args); ret.Type == VAR_NUMBER {
					if f := hook.Ret; f&0x02 != 0 {
						v := float32(ret.Value.(float64))
						WriteProcessMemory(hook.ProcessHandle, uintptr(hook.RetAddress), uintptr(unsafe.Pointer(&v)), 4)
					} else if f&0x04 != 0 {
						v := ret.Value.(float64)
						WriteProcessMemory(hook.ProcessHandle, uintptr(hook.RetAddress), uintptr(unsafe.Pointer(&v)), 8)
					} else {
						v := uint64(ret.Value.(float64))
						WriteProcessMemory(hook.ProcessHandle, uintptr(hook.RetAddress), uintptr(unsafe.Pointer(&v)), 8)
					}
				} else if ret.Type == VAR_ARRAY {
					var bytes []byte
					for _, b := range *ret.Value.(*[]Variable) {
						if b.Type == VAR_NUMBER {
							bytes = append(bytes, byte(b.Value.(float64)))
						}
					}

					l := len(bytes)

					if f := hook.Ret; f&0x02 != 0 && l == 4 {
						WriteProcessMemory(hook.ProcessHandle, uintptr(hook.RetAddress), uintptr(unsafe.Pointer(&bytes[0])), 4)
					} else if f&0x04 != 0 && l == 8 {
						WriteProcessMemory(hook.ProcessHandle, uintptr(hook.RetAddress), uintptr(unsafe.Pointer(&bytes[0])), 8)
					} else if l > 0 && l < 9 {
						WriteProcessMemory(hook.ProcessHandle, uintptr(hook.RetAddress), uintptr(unsafe.Pointer(&bytes[0])), uint(l))
					}
				}

				ResumeThread(thread)
				w32.CloseHandle(thread)
			}
		}
		return 0
	})

	w32.CloseHandle(CreateThread(syscall.NewCallback(func(ptr uintptr) uintptr {
		msgThreadId = windows.GetCurrentThreadId()
		w32.SetWindowsHookEx(w32.WH_KEYBOARD_LL, KeyboardHook, 0, 0)

		var msg w32.MSG
		for w32.GetMessage(&msg, 0, 0, 0) != 0 {
			if msg.Message == 660 {
				msgHooksMutex.RLock()
				for _, hook := range msgHooks {
					if hook.ProcessId == uint32(msg.WParam) && hook.Address == uint64(msg.LParam) {
						w32.CloseHandle(CreateThread(msgHookCallback, uintptr(unsafe.Pointer(&hook))))
						break
					}
				}
				msgHooksMutex.RUnlock()
			}

			w32.TranslateMessage(&msg)
			w32.DispatchMessage(&msg)
		}

		return 0
	}), 0))
}
