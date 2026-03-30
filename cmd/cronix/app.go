package main

import "fmt"

func Run(args []string) int {
	cmd := newRootCmd()
	cmd.SetArgs(args[1:])
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		return 1
	}
	return 0
}
