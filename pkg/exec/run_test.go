package exec

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/phpboyscout/zltest"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestRunStack(t *testing.T) {
	logHelper := zltest.New(t)
	log.Logger = zerolog.New(logHelper)
	cases := []struct {
		label        string
		errExpected  bool
		pipe         []Pipe
		logs         []string
		conditionals []string
	}{
		{
			label: "with single command and arguments that succeeds",
			pipe: []Pipe{
				{Cmd: "./testdata/test_args_2_stdout_exit_0.sh", Args: []string{"Cupcake", "ipsum", "dolor", "sit"}},
			},
			logs: []string{"\t| Cupcake ipsum dolor sit"}},
		{
			label: "with 2 commands succeeding",
			pipe: []Pipe{
				{Cmd: "cat", Args: []string{"./testdata/ipsum.txt"}},
				{Cmd: "./testdata/test_stdin_2_stdout_exit_0.sh"}},
			logs: []string{
				"\t| piping output to next command",
				"\t| donut. Ice cream icing cookie marshmallow powder sesame snaps sweet sugar",
			}},
		{
			label: "with 4 commands succeeding",
			pipe: []Pipe{
				{Cmd: "./testdata/test_args_2_stdout_exit_0.sh", Args: []string{"My name is..."}},
				{Cmd: "./testdata/test_stdin_2_stdout_exit_0.sh"},
				{Cmd: "wc"},
				{Cmd: "xargs"}},
			logs: []string{
				"\t| piping output to next command",
				"\t| 1 3 14",
			}},
		{label: "with no commands", errExpected: true, pipe: []Pipe{}},
		{label: "with 1 command that fails", errExpected: true, pipe: []Pipe{{Cmd: "./testdata/test_foo_2_stderr_exit_1.sh", Args: []string{}}}},
		{
			label:        "with one command that fails but a conditional that identifies success",
			errExpected:  false,
			pipe:         []Pipe{{Cmd: "./testdata/test_foo_2_stderr_exit_1.sh", Args: []string{}}},
			conditionals: []string{".*foo"}},
		{
			label:       "with 3 commands and the 1st one failing",
			errExpected: true,
			pipe: []Pipe{
				{Cmd: "./testdata/test_foo_2_stderr_exit_1.sh"},
				{Cmd: "./testdata/test_stdin_2_stdout_exit_0.sh"},
				{Cmd: "wc"}}},
		{
			label:       "with 3 commands and the 2nd one failing",
			errExpected: true,
			pipe: []Pipe{
				{Cmd: "cat", Args: []string{"./testdata/ipsum.txt"}},
				{Cmd: "./testdata/test_foo_2_stderr_exit_1.sh"},
				{Cmd: "wc"}}},
		/// negative logs

		{
			label:       "with a program that doesnt exist",
			errExpected: true, pipe: []Pipe{
				{Cmd: "ls", Args: []string{"-al", "/"}},
				{Cmd: "programdoesntexist", Args: []string{`--random`}}}},
	}

	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			logHelper.Reset()
			err := RunStack(CreateRunStackWithArgs(c.pipe, "."), c.conditionals...)
			if c.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, l := range c.logs {
				logHelper.Entries().ExpMsg(l)
			}
		})
	}
}

func TestCreateRunStack(t *testing.T) {
	cases := []struct {
		errExpected bool
		pipe        []string
	}{
		{pipe: []string{"ls -al /", "grep -i etc"}},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("with %d command/s in pipe and errExpected = %t", len(c.pipe), c.errExpected), func(t *testing.T) {
			stack := CreateRunStack(c.pipe, ".")
			assert.Len(t, stack, len(c.pipe))
		})
	}
}

func TestGetConditionalCheck(t *testing.T) {
	cases := []struct {
		patterns []string
		line     string
		expected bool
	}{
		{patterns: []string{"line"}, line: "This is a line of text", expected: true},
		{patterns: []string{"moo"}, line: "This is a line of text", expected: false},
		{patterns: []string{"moo", "line"}, line: "This is a line of text", expected: true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("testing for %s", strings.Join(c.patterns, " & ")), func(t *testing.T) {
			check := GetConditionalCheck(c.patterns...)
			result := check(c.line)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestHandleOutput(t *testing.T) {
	contents := `Cupcake ipsum dolor sit amet candy oat cake. Toffee tart lollipop pie
chocolate bar. Tart marzipan chocolate cake croissant gingerbread liquorice
sugar plum icing. Sweet roll marzipan halvah cookie cake gingerbread marzipan.
Ice cream gingerbread wafer chocolate cake cake carrot cake dragée cake wafer.
Candy donut powder sweet roll cookie bear claw. Gummies tiramisu bear claw


candy jujubes. Carrot cake jujubes cupcake chocolate chocolate bar oat cake
pie. Jujubes chupa chups tootsie roll brownie donut cupcake fruitcake lemon
drops. Chocolate cake pudding carrot cake tootsie roll marshmallow. Chupa chups
croissant toffee candy canes sweet tootsie roll. Cookie soufflé dessert cupcake
croissant brownie sweet roll. Cupcake jelly topping chocolate bar pudding
donut. Ice cream icing cookie marshmallow powder sesame snaps sweet sugar
plum pie.`
	cases := []struct {
		patterns []string
		line     string
		expected bool
	}{
		{patterns: []string{"cake(.*)carrot"}, expected: true},
		{patterns: []string{"marsbar"}, expected: false},
		{patterns: []string{"marsbar", "cake(.*)carrot"}, expected: true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("testing for %s", strings.Join(c.patterns, " & ")), func(t *testing.T) {
			check := GetConditionalCheck(c.patterns...)
			r := io.NopCloser(io.LimitReader(strings.NewReader(contents), int64(len(contents))))
			ch := make(chan bool)

			go func() {
				for {
					result := <-ch
					assert.Equal(t, c.expected, result)
				}
			}()

			HandleOutput(r, check, ch)

			// add a small buffer to allow the assertions to catch up with the handler
			<-time.After(30 * time.Millisecond)
		})
	}
}
