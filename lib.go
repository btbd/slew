package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/JamesHovious/w32"
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/unicode"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func global_len(this *Variable, args []Variable) Variable {
	if len(args) > 0 {
		switch args[0].Type {
		case VAR_STRING:
			return MakeVariable(VAR_NUMBER, float64(len(args[0].Value.(string))))
		case VAR_ARRAY:
			return MakeVariable(VAR_NUMBER, float64(len(*args[0].Value.(*[]Variable))))
		case VAR_OBJECT:
			return MakeVariable(VAR_NUMBER, float64(len(*args[0].Value.(*map[string]*Variable))))
		}
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
			for i := 2; i < len(args); i++ {
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
	r := console_Print(this, args)
	fmt.Println()
	return r
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
		s := args[0].Value.(string)
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

		s := args[0].Value.(string)
		l := len(s)

		if high < 1 {
			high = l + high
		}

		if low < high && low > -1 && high <= l {
			return MakeVariable(VAR_STRING, s[low:high])
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
	case int:
		return MakeVariable(VAR_NUMBER, float64(v.(int)))
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
var procIsWow64Process, procReadProcessMemory, procWriteProcessMemory, procMapVirtualKey, procTimeGetTime *syscall.Proc

func IsErrSuccess(err error) bool {
	if e, ok := err.(syscall.Errno); ok {
		if e == 0 {
			return true
		}
	}

	return false
}

func IsWow64Process(hProcess w32.HANDLE, out *bool) {
	procIsWow64Process.Call(uintptr(hProcess), uintptr(unsafe.Pointer(out)))
}

func ReadProcessMemory(hProcess w32.HANDLE, lpBaseAddress uintptr, size uint) (data []byte, err error) {
	data = make([]byte, size)
	_, _, err = procReadProcessMemory.Call(uintptr(hProcess), lpBaseAddress, uintptr(unsafe.Pointer(&data[0])), uintptr(size), uintptr(0))
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

func TimeGetTime() uint {
	r, _, err := procTimeGetTime.Call()
	if !IsErrSuccess(err) {
		return 0
	}

	return uint(r)
}

func WCharToString(wchar []uint16) string {
	if len(wchar) > 0 {
		out, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder().Bytes(*(*[]byte)(unsafe.Pointer(&wchar)))
		if err == nil {
			if i := bytes.IndexByte(out, 0); i > -1 {
				return string(out[:i])
			}
		}
	}

	return ""
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
/*                    process                  */
/***********************************************/
func process_Open(this *Variable, args []Variable) Variable {
	p := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	id := -1
	handle := w32.HANDLE(0)
	wow64 := false

	if len(args) > 0 {
		if v := args[0]; v.Type == VAR_STRING {
			snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
			if err == nil {
				var entry windows.ProcessEntry32
				entry.Size = uint32(unsafe.Sizeof(entry))

				if windows.Process32First(snapshot, &entry) == nil {
					var c error
					for c == nil {
						if strings.EqualFold(WCharToString(entry.ExeFile[:]), v.Value.(string)) {
							id = int(entry.ProcessID)
							AddProp(&p, "Size", MakeVariable(VAR_NUMBER, float64(entry.Size)))
							AddProp(&p, "Usage", MakeVariable(VAR_NUMBER, float64(entry.Usage)))
							AddProp(&p, "ModuleId", MakeVariable(VAR_NUMBER, float64(entry.ModuleID)))
							AddProp(&p, "Threads", MakeVariable(VAR_NUMBER, float64(entry.Threads)))
							AddProp(&p, "ParentId", MakeVariable(VAR_NUMBER, float64(entry.ParentProcessID)))
							AddProp(&p, "PriClassBase", MakeVariable(VAR_NUMBER, float64(entry.PriClassBase)))
							AddProp(&p, "Flags", MakeVariable(VAR_NUMBER, float64(entry.Flags)))
							AddProp(&p, "Name", MakeVariable(VAR_STRING, WCharToString(entry.ExeFile[:])))
							break
						}

						c = windows.Process32Next(snapshot, &entry)
					}
				}

				windows.CloseHandle(snapshot)
			}
		} else if v.Type == VAR_NUMBER {
			id = int(v.Value.(float64))
		}

		if id != -1 {
			handle, _ = w32.OpenProcess(w32.PROCESS_ALL_ACCESS, false, uint32(id))
			if handle != 0 {
				IsWow64Process(handle, &wow64)
			}
		}
	}

	AddProp(&p, "Handle", MakeVariable(VAR_NUMBER, float64(handle)))
	AddProp(&p, "Id", MakeVariable(VAR_NUMBER, float64(id)))
	if wow64 {
		AddProp(&p, "Wow64", MakeVariable(VAR_NUMBER, float64(1)))
		AddProp(&p, "ReadPointer", MakeVariable(VAR_NFUNCTION, process_ReadPointer32))
	} else {
		AddProp(&p, "Wow64", MakeVariable(VAR_NUMBER, float64(0)))
		AddProp(&p, "ReadPointer", MakeVariable(VAR_NFUNCTION, process_ReadPointer64))
	}
	AddProp(&p, "Read", MakeVariable(VAR_NFUNCTION, process_Read))
	AddProp(&p, "Write", MakeVariable(VAR_NFUNCTION, process_Write))
	AddProp(&p, "ReadInt", MakeVariable(VAR_NFUNCTION, process_ReadInt))
	AddProp(&p, "WriteInt", MakeVariable(VAR_NFUNCTION, process_WriteInt))
	AddProp(&p, "ReadInt32", MakeVariable(VAR_NFUNCTION, process_ReadInt))
	AddProp(&p, "WriteInt32", MakeVariable(VAR_NFUNCTION, process_WriteInt))
	AddProp(&p, "ReadInt64", MakeVariable(VAR_NFUNCTION, process_ReadInt64))
	AddProp(&p, "WriteInt64", MakeVariable(VAR_NFUNCTION, process_WriteInt64))
	AddProp(&p, "ReadUint", MakeVariable(VAR_NFUNCTION, process_ReadUint))
	AddProp(&p, "WriteUint", MakeVariable(VAR_NFUNCTION, process_WriteUint))
	AddProp(&p, "ReadUint32", MakeVariable(VAR_NFUNCTION, process_ReadUint))
	AddProp(&p, "WriteUint32", MakeVariable(VAR_NFUNCTION, process_WriteUint))
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
	AddProp(&p, "Close", MakeVariable(VAR_NFUNCTION, process_Close))
	AddProp(&p, "Kill", MakeVariable(VAR_NFUNCTION, process_Kill))
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
				AddProp(&p, "Size", MakeVariable(VAR_NUMBER, float64(entry.Size)))
				AddProp(&p, "Usage", MakeVariable(VAR_NUMBER, float64(entry.Usage)))
				AddProp(&p, "Id", MakeVariable(VAR_NUMBER, float64(entry.ProcessID)))
				AddProp(&p, "ModuleId", MakeVariable(VAR_NUMBER, float64(entry.ModuleID)))
				AddProp(&p, "Threads", MakeVariable(VAR_NUMBER, float64(entry.Threads)))
				AddProp(&p, "ParentId", MakeVariable(VAR_NUMBER, float64(entry.ParentProcessID)))
				AddProp(&p, "PriClassBase", MakeVariable(VAR_NUMBER, float64(entry.PriClassBase)))
				AddProp(&p, "Flags", MakeVariable(VAR_NUMBER, float64(entry.Flags)))
				AddProp(&p, "Name", MakeVariable(VAR_STRING, WCharToString(entry.ExeFile[:])))
				r = append(r, p)
				c = windows.Process32Next(snapshot, &entry)
			}
		}

		windows.CloseHandle(snapshot)
	}

	return MakeVariable(VAR_ARRAY, &r)
}

func GetProcessHandle(this *Variable) (handle w32.HANDLE) {
	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Handle"]; ok && v.Type == VAR_NUMBER {
				handle = w32.HANDLE(v.Value.(float64))
			}
		}
	}

	return
}

func process_Modules(this *Variable, args []Variable) Variable {
	var modules []Variable

	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Id"]; ok && v.Type == VAR_NUMBER {
				snapshot := w32.CreateToolhelp32Snapshot(w32.TH32CS_SNAPMODULE|w32.TH32CS_SNAPMODULE32, uint32(v.Value.(float64)))
				if snapshot != 0 {
					var entry w32.MODULEENTRY32
					entry.Size = uint32(unsafe.Sizeof(entry))

					if w32.Module32First(snapshot, &entry) {
						c := true
						for c {
							m := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
							AddProp(&m, "Size", MakeVariable(VAR_NUMBER, float64(entry.Size)))
							AddProp(&m, "Base", MakeVariable(VAR_NUMBER, float64(uintptr(unsafe.Pointer(entry.ModBaseAddr)))))
							AddProp(&m, "Name", MakeVariable(VAR_STRING, WCharToString(entry.SzModule[:])))
							modules = append(modules, m)
							c = w32.Module32Next(snapshot, &entry)
						}
					}

					w32.CloseHandle(snapshot)
				}
			}
		}
	}

	return MakeVariable(VAR_ARRAY, &modules)
}

func process_Close(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 {
		w32.CloseHandle(h)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Kill(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 {
		w32.TerminateProcess(h, 0)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadPointer32(this *Variable, args []Variable) Variable {
	var base uint32

	if h := GetProcessHandle(this); h != 0 {
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

	if h := GetProcessHandle(this); h != 0 {
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

func process_Read(this *Variable, args []Variable) Variable {
	var bytes []Variable

	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
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
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_ARRAY {
		var b []byte

		for _, e := range *args[1].Value.(*[]Variable) {
			if e.Type != VAR_NUMBER {
				return MakeVariable(VAR_NUMBER, float64(0))
			}

			b = append(b, byte(e.Value.(float64)))
		}

		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b[0])), uint(len(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadInt(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*int32)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteInt(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := int(args[1].Value.(float64))
		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadInt64(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*int64)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteInt64(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := int64(args[1].Value.(float64))
		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadUint(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*uint32)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteUint(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := uint(args[1].Value.(float64))
		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadUint64(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*uint64)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteUint64(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := uint64(args[1].Value.(float64))
		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadFloat(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, float64(*(*float32)(unsafe.Pointer(&data[0]))))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteFloat(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := float32(args[1].Value.(float64))
		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_ReadDouble(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_NUMBER {
		data, err := ReadProcessMemory(h, uintptr(args[0].Value.(float64)), 4)
		if err == nil {
			return MakeVariable(VAR_NUMBER, *(*float64)(unsafe.Pointer(&data[0])))
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_WriteDouble(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 1 && args[0].Type == VAR_NUMBER && args[1].Type == VAR_NUMBER {
		b := args[1].Value.(float64)
		WriteProcessMemory(h, uintptr(args[0].Value.(float64)), uintptr(unsafe.Pointer(&b)), uint(unsafe.Sizeof(b)))
	}

	return MakeVariable(VAR_NUMBER, float64(0))
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
			CallUserFunc(input_OnKeyDownFunc, []Variable{MakeVariable(VAR_NUMBER, float64(keycode))})
		} else if wParam == w32.WM_KEYUP && input_OnKeyUpFunc != nil {
			CallUserFunc(input_OnKeyUpFunc, []Variable{MakeVariable(VAR_NUMBER, float64(keycode))})
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

/* helper functions */

func CallUserFunc(v *Variable, args []Variable) {
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

	for _, e := range f[1].C {
		if v := Eval(e, &thread, stack); v.Type == VAR_RETURN {
			break
		}
	}

	StackRemove(&thread)
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

	object := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&object, "Keys", MakeVariable(VAR_NFUNCTION, object_Keys))
	StackPush("object", nil, -1, object)

	array := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&array, "Push", MakeVariable(VAR_NFUNCTION, array_Push))
	AddProp(&array, "Pop", MakeVariable(VAR_NFUNCTION, array_Pop))
	AddProp(&array, "Insert", MakeVariable(VAR_NFUNCTION, array_Insert))
	AddProp(&array, "Remove", MakeVariable(VAR_NFUNCTION, array_Remove))
	StackPush("array", nil, -1, array)

	console := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&console, "Print", MakeVariable(VAR_NFUNCTION, console_Print))
	AddProp(&console, "Println", MakeVariable(VAR_NFUNCTION, console_Println))
	AddProp(&console, "ReadLine", MakeVariable(VAR_NFUNCTION, console_ReadLine))
	AddProp(&console, "Clear", MakeVariable(VAR_NFUNCTION, console_Clear))
	StackPush("console", nil, -1, console)

	number := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&number, "ToString", MakeVariable(VAR_NFUNCTION, number_ToString))
	StackPush("number", nil, -1, number)

	str := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&str, "ToNumber", MakeVariable(VAR_NFUNCTION, string_ToNumber))
	AddProp(&str, "FromCharCode", MakeVariable(VAR_NFUNCTION, string_FromCharCode))
	AddProp(&str, "CharCodeAt", MakeVariable(VAR_NFUNCTION, string_CharCodeAt))
	AddProp(&str, "Contains", MakeVariable(VAR_NFUNCTION, string_Contains))
	AddProp(&str, "IndexOf", MakeVariable(VAR_NFUNCTION, string_IndexOf))
	AddProp(&str, "LastIndexOf", MakeVariable(VAR_NFUNCTION, string_LastIndexOf))
	AddProp(&str, "Replace", MakeVariable(VAR_NFUNCTION, string_Replace))
	AddProp(&str, "Slice", MakeVariable(VAR_NFUNCTION, string_Slice))
	AddProp(&str, "Split", MakeVariable(VAR_NFUNCTION, string_Split))
	AddProp(&str, "ToUpper", MakeVariable(VAR_NFUNCTION, string_ToUpper))
	AddProp(&str, "ToLower", MakeVariable(VAR_NFUNCTION, string_ToLower))
	AddProp(&str, "Trim", MakeVariable(VAR_NFUNCTION, string_Trim))
	StackPush("string", nil, -1, str)

	math_ := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&math_, "E", MakeVariable(VAR_NUMBER, math.E))
	AddProp(&math_, "LN2", MakeVariable(VAR_NUMBER, math.Ln2))
	AddProp(&math_, "LN10", MakeVariable(VAR_NUMBER, math.Ln10))
	AddProp(&math_, "LOG2E", MakeVariable(VAR_NUMBER, math.Log2E))
	AddProp(&math_, "LOG10E", MakeVariable(VAR_NUMBER, math.Log10E))
	AddProp(&math_, "PI", MakeVariable(VAR_NUMBER, math.Pi))
	AddProp(&math_, "PHI", MakeVariable(VAR_NUMBER, math.Phi))
	AddProp(&math_, "SQRT1_2", MakeVariable(VAR_NUMBER, math.Sqrt(1.0/2.0)))
	AddProp(&math_, "SQRT2", MakeVariable(VAR_NUMBER, math.Sqrt2))
	AddProp(&math_, "DEG_RAD", MakeVariable(VAR_NUMBER, 180/math.Pi))
	AddProp(&math_, "RAD_DEG", MakeVariable(VAR_NUMBER, math.Pi/180))
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
	AddProp(&math_, "Log1p", MakeVariable(VAR_NFUNCTION, math_Log1p))
	AddProp(&math_, "Log10", MakeVariable(VAR_NFUNCTION, math_Log10))
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
	procIsWow64Process, _ = dll.FindProc("IsWow64Process")
	procReadProcessMemory, _ = dll.FindProc("ReadProcessMemory")
	procWriteProcessMemory, _ = dll.FindProc("WriteProcessMemory")
	dll.Release()

	dll = syscall.MustLoadDLL("user32.dll")
	procMapVirtualKey, _ = dll.FindProc("MapVirtualKeyW")
	dll.Release()

	dll = syscall.MustLoadDLL("winmm.dll")
	procTimeGetTime, _ = dll.FindProc("timeGetTime")
	dll.Release()

	date := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&date, "Now", MakeVariable(VAR_NFUNCTION, date_Now))
	AddProp(&date, "Time", MakeVariable(VAR_NFUNCTION, date_Time))
	StackPush("date", nil, -1, date)

	process := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&process, "Open", MakeVariable(VAR_NFUNCTION, process_Open))
	AddProp(&process, "List", MakeVariable(VAR_NFUNCTION, process_List))
	StackPush("process", nil, -1, process)

	input := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&input, "OnKeyDown", MakeVariable(VAR_NFUNCTION, input_OnKeyDown))
	AddProp(&input, "OnKeyUp", MakeVariable(VAR_NFUNCTION, input_OnKeyUp))
	AddProp(&input, "KeyDown", MakeVariable(VAR_NFUNCTION, input_KeyDown))
	AddProp(&input, "KeyUp", MakeVariable(VAR_NFUNCTION, input_KeyUp))
	AddProp(&input, "IsKeyDown", MakeVariable(VAR_NFUNCTION, input_IsKeyDown))
	StackPush("input", nil, -1, input)

	go func() {
		w32.SetWindowsHookEx(w32.WH_KEYBOARD_LL, KeyboardHook, 0, 0)

		var msg w32.MSG
		for w32.GetMessage(&msg, 0, 0, 0) != 0 {
			w32.TranslateMessage(&msg)
			w32.DispatchMessage(&msg)
		}
	}()
}
