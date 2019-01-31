package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/JamesHovious/w32"
	"golang.org/x/sys/windows"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"sort"
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

func global_type(this *Variable, args []Variable) Variable {
	s := "unknown"

	if len(args) > 0 {
		switch args[0].Type {
		case VAR_NUMBER:
			s = "number"
			break
		case VAR_STRING:
			s = "string"
			break
		case VAR_FUNCTION:
		case VAR_NFUNCTION:
			s = "function"
			break
		case VAR_ARRAY:
			s = "array"
			break
		case VAR_OBJECT:
			s = "object"
			break
		}
	}

	return MakeVariable(VAR_STRING, s)
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

func array_Sort(this *Variable, args []Variable) Variable {
	if l := len(args); l > 1 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_FUNCTION {
		a := *args[0].Value.(*[]Variable)
		f := args[1]
		sort.SliceStable(a, func(i int, j int) bool {
			v := CallUserFunc(&f, []Variable{a[i], a[j]})
			return !(v.Type == VAR_NUMBER && v.Value.(float64) > 0)
		})
	}
	return MakeVariable(VAR_NUMBER, float64(0))
}

func array_Find(this *Variable, args []Variable) Variable {
	if len(args) > 1 && args[0].Type == VAR_ARRAY && args[1].Type == VAR_FUNCTION {
		f := args[1]
		for _, v := range *args[0].Value.(*[]Variable) {
			if r := CallUserFunc(&f, []Variable{v}); r.Type != VAR_NUMBER || r.Value.(float64) != 0 {
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
			if r := CallUserFunc(&f, []Variable{v, MakeVariable(VAR_NUMBER, float64(i))}); r.Type != VAR_NUMBER || r.Value.(float64) != 0 {
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
var procCreateRemoteThread, procGetCurrentProcessId, procIsWow64Process, procReadProcessMemory, procWriteProcessMemory, procMapVirtualKey, procVkKeyScanA, procTimeGetTime, procNtSuspendProcess, procNtResumeProcess, procVirtualQueryEx, procOpenThread, procSuspendThread, procResumeThread, procThread32First, procThread32Next, procWow64GetThreadContext, procWow64GetThreadSelectorEntry, procGetThreadTimes, procNtQueryInformationThread *syscall.Proc

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
							AddProp(&p, "Size", MakeVariable(VAR_NUMBER, float64(entry.Size)))
							AddProp(&p, "Usage", MakeVariable(VAR_NUMBER, float64(entry.Usage)))
							AddProp(&p, "ModuleId", MakeVariable(VAR_NUMBER, float64(entry.ModuleID)))
							AddProp(&p, "Threads", MakeVariable(VAR_NUMBER, float64(entry.Threads)))
							AddProp(&p, "ParentId", MakeVariable(VAR_NUMBER, float64(entry.ParentProcessID)))
							AddProp(&p, "PriClassBase", MakeVariable(VAR_NUMBER, float64(entry.PriClassBase)))
							AddProp(&p, "Flags", MakeVariable(VAR_NUMBER, float64(entry.Flags)))
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
			}
		}
	}

	AddProp(&p, "Handle", MakeVariable(VAR_NUMBER, float64(handle)))
	AddProp(&p, "Id", MakeVariable(VAR_NUMBER, float64(id)))
	AddProp(&p, "Suspend", MakeVariable(VAR_NFUNCTION, process_Suspend))
	AddProp(&p, "Resume", MakeVariable(VAR_NFUNCTION, process_Resume))
	if wow64 {
		AddProp(&p, "Wow64", MakeVariable(VAR_NUMBER, float64(1)))
		AddProp(&p, "ReadPointer", MakeVariable(VAR_NFUNCTION, process_ReadPointer32))
		AddProp(&p, "LoadLibrary", MakeVariable(VAR_NFUNCTION, process_LoadLibrary32))

	} else {
		AddProp(&p, "Wow64", MakeVariable(VAR_NUMBER, float64(0)))
		AddProp(&p, "ReadPointer", MakeVariable(VAR_NFUNCTION, process_ReadPointer64))
		AddProp(&p, "LoadLibrary", MakeVariable(VAR_NFUNCTION, process_LoadLibrary64))
	}
	AddProp(&p, "FindPattern", MakeVariable(VAR_NFUNCTION, process_FindPattern))
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
	AddProp(&p, "Threads", MakeVariable(VAR_NFUNCTION, process_Threads))
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
				AddProp(&p, "Name", MakeVariable(VAR_STRING, windows.UTF16ToString(entry.ExeFile[:])))
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

func process_Suspend(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 {
		NtSuspendProcess(h)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Resume(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 {
		NtResumeProcess(h)
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_Modules(this *Variable, args []Variable) Variable {
	var modules []Variable

	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Id"]; ok && v.Type == VAR_NUMBER {
				id := uint32(v.Value.(float64))
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
		}
	}

	return MakeVariable(VAR_ARRAY, &modules)
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

	if process := GetProcessHandle(this); process != 0 {
		m := *(*this).Value.(*map[string]*Variable)
		if v, ok := m["Id"]; ok && v.Type == VAR_NUMBER {
			id := w32.DWORD(v.Value.(float64))
			snapshot := w32.CreateToolhelp32Snapshot(w32.TH32CS_SNAPTHREAD, 0)
			if snapshot != 0 {
				var entry THREADENTRY32
				entry.DwSize = w32.DWORD(unsafe.Sizeof(entry))

				if Thread32First(snapshot, &entry) {
					c := true
					for c {
						if entry.Th32OwnerProcessID == id {
							t := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
							AddProp(&t, "Id", MakeVariable(VAR_NUMBER, float64(entry.Th32ThreadID)))
							AddProp(&t, "Owner", MakeVariable(VAR_NUMBER, float64(entry.Th32ThreadID)))
							AddProp(&t, "Priority", MakeVariable(VAR_NUMBER, float64(entry.TpBasePri)))

							time := int64(0)
							stack := uint64(0)
							if handle := OpenThread(0x0008|0x0002|0x0040, w32.BOOL(0), entry.Th32ThreadID); handle != 0 {
								time = GetThreadCreationTime(handle)
								if v, ok := m["Wow64"]; ok && v.Type == VAR_NUMBER {
									if i := int(v.Value.(float64)); i == 0 || i == 1 {
										funcs := [](func(w32.HANDLE, w32.DWORD, w32.HANDLE) uint64){GetThreadStack64, GetThreadStack32}
										stack = funcs[i](process, id, handle)
									}
								}
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
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_STRING {
		if v, ok := (*(*this).Value.(*map[string]*Variable))["Id"]; ok && v.Type == VAR_NUMBER {
			path := make([]uint16, 0xFF)
			windows.GetFullPathName(windows.StringToUTF16Ptr(args[0].Value.(string)), 0xFF, (*uint16)(unsafe.Pointer(&path[0])), nil)

			mod := GetModuleInfoByName(uint32(v.Value.(float64)), "kernelbase.dll")
			proc := uint32(FindPattern(h, []byte{0x83, 0xEC, 0x1C, 0x53, 0x56, 0x57, 0x85, 0xC0}, []byte("xxxxxxxx"), uint64(uintptr(unsafe.Pointer(mod.ModBaseAddr))), uint64(mod.ModBaseSize)))

			if proc != 0 {
				proc -= 17
				if WriteProcessMemory(h, uintptr(proc), uintptr(unsafe.Pointer(&([]byte{0x8B, 0xFF, 0x55, 0x8B, 0xEC, 0x5D}[0]))), 6) == nil {
					if arg, err := w32.VirtualAllocEx(h, 0, 510, w32.MEM_RESERVE|w32.MEM_COMMIT, w32.PAGE_EXECUTE_READWRITE); err == nil && WriteProcessMemory(h, arg, uintptr(unsafe.Pointer(&path[0])), 510) == nil {
						if stub, err := w32.VirtualAllocEx(h, 0, 0xFF, w32.MEM_RESERVE|w32.MEM_COMMIT, w32.PAGE_EXECUTE_READWRITE); err == nil {
							b := []byte{0x6A, 0x00, 0x6A, 0x00, 0x68, 0x00, 0x00, 0x00, 0x00, 0xE8, 0x00, 0x00, 0x00, 0x00, 0xC3}
							*(*uint32)(unsafe.Pointer(&b[5])) = uint32(arg)
							*(*uint32)(unsafe.Pointer(&b[10])) = uint32(proc) - (uint32(stub) + 14)
							if WriteProcessMemory(h, stub, uintptr(unsafe.Pointer(&b[0])), uint(len(b))) == nil {
								if thread := CreateRemoteThread(h, stub, 0); thread != 0 {
									windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)
									w32.CloseHandle(thread)
								}
							}
							w32.VirtualFreeEx(h, stub, 0, w32.MEM_RELEASE|w32.MEM_DECOMMIT)
						}
						w32.VirtualFreeEx(h, arg, 0, w32.MEM_RELEASE|w32.MEM_DECOMMIT)
					}
				}
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func process_LoadLibrary64(this *Variable, args []Variable) Variable {
	if h := GetProcessHandle(this); h != 0 && len(args) > 0 && args[0].Type == VAR_STRING {
		path := make([]uint16, 0xFF)
		windows.GetFullPathName(windows.StringToUTF16Ptr(args[0].Value.(string)), 0xFF, (*uint16)(unsafe.Pointer(&path[0])), nil)

		dll := syscall.MustLoadDLL("kernel32.dll")
		addr, _ := windows.GetProcAddress(windows.Handle(dll.Handle), "LoadLibraryW")
		dll.Release()

		arg, err := w32.VirtualAllocEx(h, 0, 510, w32.MEM_RESERVE|w32.MEM_COMMIT, w32.PAGE_EXECUTE_READWRITE)
		if err == nil && WriteProcessMemory(h, arg, uintptr(unsafe.Pointer(&path[0])), 510) == nil {
			if thread := CreateRemoteThread(h, addr, arg); thread != 0 {
				windows.WaitForSingleObject(windows.Handle(thread), w32.INFINITE)
				w32.VirtualFreeEx(h, arg, 0, w32.MEM_RELEASE|w32.MEM_DECOMMIT)
				w32.CloseHandle(thread)
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func thread_Suspend(this *Variable, args []Variable) Variable {
	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Id"]; ok && v.Type == VAR_NUMBER {
				if handle := OpenThread(0x0002, w32.BOOL(0), w32.DWORD(v.Value.(float64))); handle != 0 {
					SuspendThread(handle)
					w32.CloseHandle(handle)
				}
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
}

func thread_Resume(this *Variable, args []Variable) Variable {
	if this != nil {
		if p := *this; p.Type == VAR_OBJECT {
			if v, ok := (*p.Value.(*map[string]*Variable))["Id"]; ok && v.Type == VAR_NUMBER {
				if handle := OpenThread(0x0002, w32.BOOL(0), w32.DWORD(v.Value.(float64))); handle != 0 {
					ResumeThread(handle)
					w32.CloseHandle(handle)
				}
			}
		}
	}

	return MakeVariable(VAR_NUMBER, float64(0))
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
		if mi.State&(w32.MEM_COMMIT|w32.MEM_RESERVE) != 0 {
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
	if h := GetProcessHandle(this); h != 0 {
		if l := len(args); l == 1 && args[0].Type == VAR_STRING {
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

/***********************************************/
/*                       proc                  */
/***********************************************/
func proc_GetAddress(this *Variable, args []Variable) Variable {
	var r uintptr

	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		dll := syscall.MustLoadDLL(args[0].Value.(string))
		r, _ = windows.GetProcAddress(windows.Handle(dll.Handle), args[1].Value.(string))
		dll.Release()
	}

	return MakeVariable(VAR_NUMBER, float64(r))
}

func proc_GetRelativeAddress(this *Variable, args []Variable) Variable {
	var r uintptr

	if len(args) > 1 && args[0].Type == VAR_STRING && args[1].Type == VAR_STRING {
		dll := syscall.MustLoadDLL(args[0].Value.(string))
		r, _ = windows.GetProcAddress(windows.Handle(dll.Handle), args[1].Value.(string))
		r -= uintptr(dll.Handle)
		dll.Release()
	}

	return MakeVariable(VAR_NUMBER, float64(r))
}

/* helper functions */
func CallUserFunc(v *Variable, args []Variable) Variable {
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

	object := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&object, "Keys", MakeVariable(VAR_NFUNCTION, object_Keys))
	StackPush("object", nil, -1, object)

	array := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&array, "Push", MakeVariable(VAR_NFUNCTION, array_Push))
	AddProp(&array, "Pop", MakeVariable(VAR_NFUNCTION, array_Pop))
	AddProp(&array, "Insert", MakeVariable(VAR_NFUNCTION, array_Insert))
	AddProp(&array, "Remove", MakeVariable(VAR_NFUNCTION, array_Remove))
	AddProp(&array, "Sort", MakeVariable(VAR_NFUNCTION, array_Sort))
	AddProp(&array, "Find", MakeVariable(VAR_NFUNCTION, array_Find))
	AddProp(&array, "Each", MakeVariable(VAR_NFUNCTION, array_Each))
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
	procGetCurrentProcessId, _ = dll.FindProc("GetCurrentProcessId")
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
	dll.Release()

	dll = syscall.MustLoadDLL("user32.dll")
	procMapVirtualKey, _ = dll.FindProc("MapVirtualKeyW")
	procVkKeyScanA, _ = dll.FindProc("VkKeyScanA")
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

	process := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&process, "Open", MakeVariable(VAR_NFUNCTION, process_Open))
	AddProp(&process, "List", MakeVariable(VAR_NFUNCTION, process_List))
	AddProp(&process, "Current", process_Open(nil, []Variable{MakeVariable(VAR_NUMBER, float64(GetCurrentProcessId()))}))
	StackPush("process", nil, -1, process)

	input := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&input, "OnKeyDown", MakeVariable(VAR_NFUNCTION, input_OnKeyDown))
	AddProp(&input, "OnKeyUp", MakeVariable(VAR_NFUNCTION, input_OnKeyUp))
	AddProp(&input, "KeyDown", MakeVariable(VAR_NFUNCTION, input_KeyDown))
	AddProp(&input, "KeyUp", MakeVariable(VAR_NFUNCTION, input_KeyUp))
	AddProp(&input, "IsKeyDown", MakeVariable(VAR_NFUNCTION, input_IsKeyDown))
	AddProp(&input, "SendKeys", MakeVariable(VAR_NFUNCTION, input_SendKeys))
	StackPush("input", nil, -1, input)

	proc := MakeVariable(VAR_OBJECT, &map[string]*Variable{})
	AddProp(&proc, "GetAddress", MakeVariable(VAR_NFUNCTION, proc_GetAddress))
	AddProp(&proc, "GetRelativeAddress", MakeVariable(VAR_NFUNCTION, proc_GetRelativeAddress))
	StackPush("proc", nil, -1, proc)

	go func() {
		w32.SetWindowsHookEx(w32.WH_KEYBOARD_LL, KeyboardHook, 0, 0)

		var msg w32.MSG
		for w32.GetMessage(&msg, 0, 0, 0) != 0 {
			w32.TranslateMessage(&msg)
			w32.DispatchMessage(&msg)
		}
	}()
}
