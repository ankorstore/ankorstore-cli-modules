package exec

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/go-errors/errors"
	"github.com/rs/zerolog/log"
)

type RunCmd struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

type Pipe struct {
	Cmd  string
	Args []string
}

// Run the supplied command.
func Run(command string, conditionals ...string) error {
	return RunDir(command, ".", conditionals...)
}

// RunDir run the supplied command from the specified directory.
func RunDir(command string, dir string, conditionals ...string) error {
	pipe := strings.Split(command, "|")
	return RunStack(CreateRunStack(pipe, dir), conditionals...)
}

// RunArgs run the command with supplied args (not safe for args that require whitespace).
func RunArgs(command string, args []string, dir string, conditionals ...string) error {
	pipe := strings.Split(fmt.Sprintf("%s %s", command, strings.Join(args, " ")), "|")
	return RunStack(CreateRunStack(pipe, dir), conditionals...)
}

// CreateRunStackWithArgs create a run stack from the slice of Pipe structs, should automatically chain the stdout
// of the previous command into the stdin of the subsequent command.
func CreateRunStackWithArgs(pipe []Pipe, dir string) []*RunCmd {
	stack := make([]*RunCmd, len(pipe))
	if len(pipe) == 0 {
		return stack
	}
	last := len(pipe) - 1

	// build our commands
	for i, c := range pipe {
		cmd := exec.Command(c.Cmd, c.Args...) //nolint: gosec
		cmd.Dir = dir

		stderrPipe, _ := cmd.StderrPipe()
		stack[i] = &RunCmd{cmd: cmd, stdout: nil, stderr: stderrPipe}
	}

	// now wire together with pipes
	for i := range stack[:last] {
		stack[i+1].cmd.Stdin, _ = stack[i].cmd.StdoutPipe()
	}

	// configure our last command in the chain
	stack[last].stdout, _ = stack[last].cmd.StdoutPipe()

	return stack
}

// CreateRunStack create a run stack from the slice of command strings.
func CreateRunStack(pipe []string, dir string) []*RunCmd {
	stack := make([]Pipe, len(pipe))

	for i, c := range pipe {
		split := strings.Split(strings.Trim(c, " "), " ")
		var parts []string
		for _, v := range split {
			if v != "" {
				parts = append(parts, v)
			}
		}
		stack[i] = Pipe{
			Cmd:  parts[0],
			Args: parts[1:],
		}
	}

	return CreateRunStackWithArgs(stack, dir)
}

// RunStack take the supplied stack of commands and run it if any conditionals are satisfied in the output.
func RunStack(stack []*RunCmd, conditionals ...string) (err error) {
	isSuccessMessageInStdOut := make(chan bool)
	isSuccessMessageInStdErr := make(chan bool)

	var successDispiteErr = false

	if len(stack) == 0 {
		err := errors.New("no run stack defined")
		log.Error().Err(err).Send()
		return err
	}

	successDespiteErrWg := &sync.WaitGroup{}
	successDespiteErrWg.Add(1)
	go func(wg *sync.WaitGroup) {
		select {
		case stdout := <-isSuccessMessageInStdOut:
			if stdout {
				successDispiteErr = true
			}
		case stderr := <-isSuccessMessageInStdErr:
			if stderr {
				successDispiteErr = true
			}
		}
		wg.Done()
	}(successDespiteErrWg)

	log.Debug().Msgf("Running: %s from %s", stack[0].cmd, stack[0].cmd.Dir)

	if stack[0].cmd.Process == nil {
		if err = stack[0].cmd.Start(); err != nil {
			e := errors.Wrap(fmt.Errorf("%s, %w", stack[0].cmd.String(), err), 0)
			log.Error().Err(e).Send()
			return e
		}
	}

	if stack[0].cmd.Process != nil {
		check := GetConditionalCheck(conditionals...)
		if stack[0].stderr != nil {
			go HandleOutput(stack[0].stderr, check, isSuccessMessageInStdOut)
		}
		if stack[0].stdout != nil {
			go HandleOutput(stack[0].stdout, check, isSuccessMessageInStdErr)
		}
	}

	if len(stack) > 1 {
		// start the next command in the chain to trigger
		// the read of the incoming pipe
		if err = stack[1].cmd.Start(); err != nil {
			return errors.Wrap(fmt.Errorf("%s, %w", stack[1].cmd.String(), err), 0)
		}
		defer func() {
			_ = stack[0].cmd.Stdout.(io.Closer).Close()
			if err == nil {
				// how handle the output and errors from the next command in the pipe
				log.Debug().Msgf("\t| piping output to next command")
				err = RunStack(stack[1:], conditionals...)
			}
		}()
	}

	result := stack[0].cmd.Wait()

	successDespiteErrWg.Wait()
	if successDispiteErr {
		return nil
	}

	if result != nil {
		e := errors.Wrap(fmt.Errorf("%s, %w", stack[0].cmd.String(), result), 0)
		log.Error().Err(e).Send()
		return e
	}

	return nil
}

// GetConditionalCheck return a function that can be used to check each line of output for content that indicates success.
func GetConditionalCheck(conditionals ...string) func(line string) bool {
	var permissibleState bool
	var permissibleConditions = make([]*regexp.Regexp, len(conditionals))
	for i, conditional := range conditionals {
		permissibleConditions[i] = regexp.MustCompile(conditional)
	}

	return func(line string) bool {
		for _, pc := range permissibleConditions {
			if pc.MatchString(line) {
				permissibleState = true
			}
		}
		return permissibleState
	}
}

// HandleOutput take the supplied output and print it to screen checking for matches in the
// output to indicate if the output indicates success.
func HandleOutput(in io.ReadCloser, permitted func(string) bool, permissible chan<- bool) {
	scanner := bufio.NewScanner(in)
	scanner.Split(bufio.ScanLines)
	var prev string
	prev = ""
	var p bool
	for scanner.Scan() {
		line := scanner.Text()
		if ok := permitted(line); ok {
			permissible <- true
			p = true
		}
		if line != prev {
			log.Debug().Msgf("\t| %s", line)
		}
		prev = line
	}
	if !p {
		permissible <- false
	}
}
