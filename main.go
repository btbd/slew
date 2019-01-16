package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	debug := 0

	usage := flag.Usage
	flag.Usage = func() {
		fmt.Println("A simple interpreted language")
		usage()
	}

	flag.IntVar(&debug, "d", debug, "debug mode (0-3)")
	flag.Parse()

	InitStack()

	var thread []Stack
	if args := flag.Args(); len(args) > 0 {
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
