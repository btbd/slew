package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	var thread []Stack
	InitStack()

	if !strings.HasSuffix(strings.ToLower(os.Args[0]), ".exe") {
		os.Args[0] += ".exe"
	}

	sig := strings.Replace("/*552B5034631B3CE91BE3F3542AFAF7BE95C4087648AA9C79632CAB699C6B0F61*/", "A", "Z", -1)
	if body, err := ioutil.ReadFile(os.Args[0]); err == nil {
		b := string(body)
		if i := strings.Index(b, sig); i > -1 {
			Eval(Parse(Tokenize(b[i:])), &thread, -1)
			os.Exit(0)
		}
	}

	debug := 0
	portable := ""

	usage := flag.Usage
	flag.Usage = func() {
		fmt.Println("A simple interpreted language")
		usage()
	}

	flag.IntVar(&debug, "d", debug, "debug mode (0-3)")
	flag.StringVar(&portable, "o", portable, "output portable name")
	flag.Parse()

	if portable != "" {
		if args := flag.Args(); len(args) > 0 {
			body, err := ioutil.ReadFile(args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			data, err := ioutil.ReadFile(os.Args[0])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			if err := ioutil.WriteFile(portable, data, 0644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			f, err := os.OpenFile(portable, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer f.Close()

			if _, err := f.WriteString(sig + string(body)); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			os.Exit(0)
		} else {
			fmt.Fprintln(os.Stderr, "error: no file specified")
			os.Exit(1)
		}
	} else if args := flag.Args(); len(args) > 0 {
		body, err := ioutil.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		tokens := Tokenize(string(body))
		if debug > 1 {
			for _, e := range tokens {
				fmt.Println(e.Type, e.Value)
			}
		}

		tree := Parse(tokens)
		if debug > 0 {
			PrintTree(tree, 0)
		}

		Eval(tree, &thread, -1)

		if debug > 0 {
			fmt.Println(global_stack, thread)
		}
	} else {
		reader := bufio.NewReader(os.Stdin)

		for {
			fmt.Print("> ")
			text, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			tokens := Tokenize(string(text))
			if debug > 1 {
				for _, e := range tokens {
					fmt.Println(e.Type, e.Value)
				}
			}

			tree := Parse(tokens)
			if debug > 0 {
				PrintTree(tree, 0)
			}

			v := ReduceVariable(Eval(tree, &thread, -1))

			if debug > 0 {
				fmt.Println(global_stack, thread)
			}

			fmt.Println("<", ToString(v))
		}
	}
}
