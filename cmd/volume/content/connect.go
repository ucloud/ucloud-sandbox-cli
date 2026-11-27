package content

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	volumecontent "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content"
	contentapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content/api"
	"golang.org/x/term"
)

type connectOperation struct{}

func (o *connectOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <volume-id>",
		Short: "Open an interactive volume command line (ls, ll, cd, cat, mkdir, rm, exit)",
		Args:  cobra.ExactArgs(1),
	}
}

func (o *connectOperation) Run(ctx cmd.OperationContext) error {
	content, err := open(ctx)
	if err != nil {
		return err
	}
	s := contentSession{ctx: ctx, content: content, cwd: "/"}
	if err := s.changeDirectory("/"); err != nil {
		return err
	}

	// Use line editing and in-memory history on a terminal; keep piped input
	// deterministic and avoid changing terminal state for redirected streams.
	if input, ok := ctx.Cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(input.Fd())) {
		stdin := readline.NewCancelableStdin(input)
		rl, err := readline.NewEx(&readline.Config{
			Stdin: stdin, Stdout: ctx.Cmd.OutOrStdout(), Stderr: ctx.Cmd.ErrOrStderr(),
			EOFPrompt: "\n", InterruptPrompt: "\n",
		})
		if err != nil {
			stdin.Close()
			return err
		}
		defer func() {
			stdin.Close()
			rl.Close()
		}()
		return s.run(func(prompt string) (string, error) {
			rl.SetPrompt(prompt)
			return rl.Readline()
		})
	}

	scanner := bufio.NewScanner(ctx.Cmd.InOrStdin())
	scanner.Buffer(make([]byte, 4096), 1<<20)
	return s.run(func(prompt string) (string, error) {
		if _, err := fmt.Fprint(ctx.Cmd.OutOrStdout(), prompt); err != nil {
			return "", err
		}
		if scanner.Scan() {
			return scanner.Text(), nil
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	})
}

type contentSession struct {
	ctx     cmd.OperationContext
	content *volumecontent.Content
	cwd     string
}

func (s *contentSession) run(readLine func(string) (string, error)) error {
	for {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		line, err := readLine(s.cwd + " > ")
		if errors.Is(err, io.EOF) || errors.Is(err, readline.ErrInterrupt) {
			return nil
		}
		if err != nil {
			return err
		}
		args, err := splitCommand(line)
		if err == nil && len(args) == 0 {
			continue
		}
		if err == nil && len(args) == 1 && args[0] == "exit" {
			return nil
		}
		if err == nil {
			err = s.execute(args)
		}
		if err != nil {
			if _, writeErr := fmt.Fprintln(s.ctx.Cmd.ErrOrStderr(), "Error:", err); writeErr != nil {
				return writeErr
			}
		}
	}
}

// resolve converts the session's absolute display paths into API paths.
// Cleaning against / keeps parent traversal confined to the volume root.
func (s *contentSession) resolve(value string) (string, error) {
	if value == "" || strings.ContainsRune(value, 0) {
		return "", fmt.Errorf("path must not be empty or contain NUL bytes")
	}
	absolute := value
	if !path.IsAbs(value) {
		absolute = path.Join(s.cwd, value)
	}
	relative := strings.TrimPrefix(path.Clean(absolute), "/")
	if relative == "" {
		return ".", nil
	}
	return relative, nil
}

func (s *contentSession) changeDirectory(value string) error {
	resolved, err := s.resolve(value)
	if err != nil {
		return err
	}
	entry, err := s.content.Stat(s.ctx, &contentapi.StatVolumePathParams{Path: resolved, FollowSymlinks: new(true)})
	if err != nil {
		return err
	}
	if entry.Type != contentapi.Directory {
		return fmt.Errorf("%s: not a directory", value)
	}
	s.cwd = path.Clean("/" + resolved)
	return nil
}

func (s *contentSession) execute(args []string) error {
	name, operands := args[0], args[1:]
	value := "."
	switch name {
	case "ls", "ll", "cd":
		if len(operands) > 1 {
			return fmt.Errorf("usage: %s [path]", name)
		}
		if len(operands) == 1 {
			value = operands[0]
		} else if name == "cd" {
			value = "/"
		}
	case "cat", "mkdir", "rm":
		if len(operands) != 1 {
			return fmt.Errorf("usage: %s <path>", name)
		}
		value = operands[0]
	default:
		return fmt.Errorf("supported commands: ls [path], ll [path], cd [path], cat <path>, mkdir <path>, rm <path>, exit")
	}
	if name == "cd" {
		return s.changeDirectory(value)
	}
	resolved, err := s.resolve(value)
	if err != nil {
		return err
	}
	switch name {
	case "ls", "ll":
		entries, err := s.content.List(s.ctx, &contentapi.ListVolumeDirectoryParams{Path: &resolved, Depth: new(int32(1))})
		if err != nil {
			return err
		}
		if name == "ll" {
			return renderEntries(s.ctx.Cmd.OutOrStdout(), entries)
		}
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name
		}
		slices.Sort(names)
		if len(names) > 0 {
			_, err = fmt.Fprintln(s.ctx.Cmd.OutOrStdout(), strings.Join(names, "  "))
		}
		return err
	case "cat":
		response, err := s.content.ReadFile(s.ctx, &contentapi.ReadVolumeFileParams{Path: resolved})
		if err != nil {
			return err
		}
		defer response.Body.Close()
		err = writeText(s.ctx.Cmd.OutOrStdout(), response.Body)
		if errors.Is(err, errBinaryContent) {
			return errors.New("file contains binary or non-UTF-8 content and cannot be displayed; use 'ucloud-sandbox-cli vol ct download <volume-id> <path> -f <local-file>'")
		}
		return err
	case "mkdir":
		_, err := s.content.MakeDir(s.ctx, contentapi.CreateDirectory{Path: resolved})
		return err
	case "rm":
		return s.content.Remove(s.ctx, &contentapi.DeleteVolumePathParams{Path: resolved})
	}
	return nil
}

// splitCommand supports quoted names and escaped characters without executing
// shell substitutions or expanding variables and wildcards.
func splitCommand(line string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	escaped, started := false, false
	for _, r := range line {
		switch {
		case escaped:
			word.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped, started = true, true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, started = r, true
		case unicode.IsSpace(r):
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
		default:
			word.WriteRune(r)
			started = true
		}
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("unfinished quote or escape")
	}
	if started {
		args = append(args, word.String())
	}
	return args, nil
}
