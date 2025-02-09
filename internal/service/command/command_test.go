package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"testing"

	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/storage"
)

type mockCommand struct {
	exitCode int
}

func (m *mockCommand) Run() error {
	if m.exitCode == 0 {
		return nil
	}
	return errors.New("Command returned an error")
}

func (m *mockCommand) StdoutPipe() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("stdout message")), nil
}

func (m *mockCommand) StderrPipe() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBufferString("stderr message")), nil
}

func (m *mockCommand) ExitCode() int {
	return m.exitCode
}

type mockCommander struct{}

func (m *mockCommander) Command(name string, arg ...string) Command {
	if arg[len(arg)-1] == "fail" {
		return &mockCommand{
			exitCode: 1,
		}
	}
	return &mockCommand{
		exitCode: 0,
	}
}

type mockStorage struct {
	commands []entity.Command
}

func (m *mockStorage) GetCommands(ctx context.Context) ([]entity.Command, error) {
	return m.commands, nil
}

func (m *mockStorage) GetCommandById(ctx context.Context, id string) (*entity.Command, error) {
	num, err := strconv.Atoi(id)
	if err != nil {
		return nil, CommandNotFoundError
	}
	if num < 0 || num >= len(m.commands) {
		return nil, CommandNotFoundError
	}
	return &m.commands[num], nil
}

func (m *mockStorage) GetUser(ctx context.Context, username string) (entity.User, error) {
	return entity.User{}, storage.UserNotFoundError
}

func (m *mockStorage) LoadConfig() error {
	return nil
}

var (
	role1    = "developer"
	role2    = "admin"
	mockCmds = []entity.Command{
		{Name: "List", Command: "ls -la"},
		{Name: "Hello", Command: "echo Hello", Role: &role1},
		{Name: "Echo secret", Command: "echo $SECRET", Role: &role2},
		{Name: "Failing", Command: "fail"},
	}
	mockSt = &mockStorage{
		commands: mockCmds,
	}
	user1     = entity.User{}
	user2     = entity.User{Roles: []string{"developer"}}
	user3     = entity.User{Roles: []string{"developer", "admin"}}
	commander = &mockCommander{}
)

func TestGetCommands(t *testing.T) {
	// Arrange
	expectedCmds := []entity.Command{
		mockCmds[0],
		mockCmds[1],
		mockCmds[3],
	}

	cs := NewCommandService(mockSt, commander)

	// Act
	cmds, err := cs.GetCommands(context.Background(), user2)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if len(cmds) != len(expectedCmds) {
		t.Fatalf("Expected %d commands, got %d", len(expectedCmds), len(cmds))
	}

	for i, cmd := range cmds {
		if cmd.Name != expectedCmds[i].Name || cmd.Command != expectedCmds[i].Command {
			t.Errorf("Expected command %v, got %v", expectedCmds[i], cmd)
		}
	}
}

func TestExecuteCommand(t *testing.T) {
	t.Run("Valid ID", func(t *testing.T) {
		// Arrange
		cs := NewCommandService(mockSt, commander)

		// Act
		execID, err := cs.ExecuteCommand(context.Background(), user1, "0")

		// Assert
		if err != nil {
			t.Fatalf("ExecuteCommand failed: %q", err)
		}

		cs.WaitExecutions(context.Background())

		exec, err := cs.GetExecution(context.Background(), user1, execID)
		if err != nil {
			t.Fatalf("Got error %q when getting execution", err)
		}
		if exec == nil {
			t.Fatalf("Expected execution with ID %d, got nil", execID)
		}

		stdoutFound := false
		stderrFound := false
		for _, logEntry := range exec.Log {
			if logEntry.Stream == "stdout" && logEntry.Data == "stdout message" {
				stdoutFound = true
			}
			if logEntry.Stream == "stderr" && logEntry.Data == "stderr message" {
				stderrFound = true
			}
		}

		if !stdoutFound {
			t.Errorf("Expected 'stdout message' in stdout log, got %v", exec.Log)
		}

		if !stderrFound {
			t.Errorf("Expected 'stderr message' in stderr log, got %v", exec.Log)
		}
	})

	t.Run("Invalid ID", func(t *testing.T) {
		// Arrange
		cs := NewCommandService(mockSt, commander)

		// Act
		_, err := cs.ExecuteCommand(context.Background(), user1, "5")

		// Assert
		if err == nil {
			t.Fatalf("Expected error for invalid command ID, got none")
		}

		if !errors.Is(err, CommandNotFoundError) {
			t.Errorf("Expected CommandNotFoundError, got %q", err)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		// Arrange
		cs := NewCommandService(mockSt, commander)

		// Act
		_, err := cs.ExecuteCommand(context.Background(), user2, "2")

		// Assert
		if err == nil {
			t.Fatalf("Expected error for invalid command ID, got none")
		}

		if !errors.Is(err, UnauthorizedError) {
			t.Errorf("Expected UnauthorizedError, got %q", err)
		}
	})

	t.Run("Command Failure", func(t *testing.T) {
		// Arrange
		cs := NewCommandService(mockSt, commander)

		// Act
		execID, err := cs.ExecuteCommand(context.Background(), user1, "3")

		// Assert
		if err != nil {
			t.Fatalf("ExecuteCommand failed: %q", err)
		}

		cs.WaitExecutions(context.Background())

		exec, err := cs.GetExecution(context.Background(), user1, execID)
		if err != nil {
			t.Fatalf("Got error %q when getting execution", err)
		}
		if exec == nil {
			t.Fatalf("Expected execution with ID %d, got nil", execID)
		}

		if exec.ExitCode == nil || *exec.ExitCode != 1 {
			t.Errorf("Expected exit code 1, got %v", exec.ExitCode)
		}
	})
}

func TestGetExecutionHistory(t *testing.T) {
	// Arrange
	expectedCommand := mockCmds[0]
	cs := NewCommandService(mockSt, commander)

	_, err := cs.ExecuteCommand(context.Background(), user1, "0")
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %q", err)
	}

	cs.WaitExecutions(context.Background())

	// Act
	history, err := cs.GetExecutionHistory(context.Background(), user1)

	// Assert
	if err != nil {
		t.Fatalf("GetExecutionHistory failed: %q", err)
	}

	if len(history) != 1 {
		t.Fatalf("Expected history length 1, got %d", len(history))
	}

	if history[0].CommandName != expectedCommand.Name {
		t.Errorf("Expected CommandName %q, got %q", expectedCommand.Name, history[0].CommandName)
	}
}

func TestGetExecution(t *testing.T) {
	// Arrange
	cs := NewCommandService(mockSt, commander)

	execID, err := cs.ExecuteCommand(context.Background(), user1, "0")
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %q", err)
	}

	cs.WaitExecutions(context.Background())

	// Act
	exec, err := cs.GetExecution(context.Background(), user1, execID)

	// Assert
	if err != nil {
		t.Fatalf("Got error %q when getting execution", err)
	}
	if exec == nil {
		t.Fatalf("Expected execution with ID %d, got nil", execID)
	}

	if exec.ExecId != execID {
		t.Errorf("Expected ExecId %d, got %d", execID, exec.ExecId)
	}

	if exec.CommandId != "0" {
		t.Errorf("Expected CommandId 0, got %s", exec.CommandId)
	}

	if exec.ExitCode == nil || *exec.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %v", exec.ExitCode)
	}

	if len(exec.Log) == 0 {
		t.Errorf("Expected log entries, got none")
	}
}
