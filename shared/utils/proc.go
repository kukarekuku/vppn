package utils

import (
	"../command"
	"errors"
	"fmt"
	"io"
	"os"
)

func Exec(name string, arg ...string) (err error) {
	cmd := command.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to exec '%s' ", name) + err.Error())
		return
	}

	return
}

func ExecInput(input, name string, arg ...string) (err error) {
	cmd := command.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to get stdin in exec '%s' ", name) + err.Error())
		return
	}
	defer stdin.Close()

	err = cmd.Start()
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to exec '%s' ", name) + err.Error())
		return
	}

	_, err = io.WriteString(stdin, input)
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to write stdin in exec '%s' ", name) + err.Error())
		return
	}

	err = cmd.Wait()
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to exec '%s' ", name) + err.Error())
		return
	}

	return
}

func ExecOutput(name string, arg ...string) (output string, err error) {
	cmd := command.Command(name, arg...)
	cmd.Stderr = os.Stderr

	outputByt, err := cmd.Output()
	if err != nil {
		err = errors.New(fmt.Sprintf("utils: Failed to exec '%s' ", name) + err.Error())
		return
	}
	output = string(outputByt)

	return
}
