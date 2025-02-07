package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jrammler/wheelhouse/internal/entity"
	"github.com/jrammler/wheelhouse/internal/storage"
)

type mockCommand struct {
	runFunc        func() error
	stdoutPipeFunc func() (io.ReadCloser, error)
	stderrPipeFunc func() (io.ReadCloser, error)
	exitCode       int
}

func (m *mockCommand) Run() error {
	if m.runFunc != nil {
		return m.runFunc()
	}
	return nil
}

func (m *mockCommand) StdoutPipe() (io.ReadCloser, error) {
	if m.stdoutPipeFunc != nil {
		return m.stdoutPipeFunc()
	}
	return io.NopCloser(bytes.NewBufferString("stdout mock output")), nil
}

func (m *mockCommand) StderrPipe() (io.ReadCloser, error) {
	if m.stderrPipeFunc != nil {
		return m.stderrPipeFunc()
	}
	return io.NopCloser(bytes.NewBufferString("stderr mock output")), nil
}

func (m *mockCommand) ExitCode() int {
	return m.exitCode
}

type mockCommander struct {
	commandFunc func(name string, arg ...string) Command
}

func (m *mockCommander) Command(name string, arg ...string) Command {
	if m.commandFunc != nil {
		return m.commandFunc(name, arg...)
	}
	return &mockCommand{
		exitCode: 0,
	}
}

type mockStorage struct {
	commands []entity.Command
	users    []entity.User
}

func (m *mockStorage) GetCommands(ctx context.Context) ([]entity.Command, error) {
	return m.commands, nil
}

func (m *mockStorage) GetUser(ctx context.Context, username string) (entity.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return entity.User{}, storage.UserNotFoundError
}

func (m *mockStorage) LoadConfig() error {
	return nil
}

func TestGetCommands(t *testing.T) {
	// Arrange
	mockCmds := []entity.Command{
		{Name: "List", Command: "ls -la"},
		{Name: "Echo", Command: "echo Hello"},
	}

	mockSt := &mockStorage{
		commands: mockCmds,
	}

	cs := NewCommandService(mockSt, &mockCommander{})

	// Act
	cmds, err := cs.GetCommands(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(cmds) != len(mockCmds) {
		t.Fatalf("Expected %d commands, got %d", len(mockCmds), len(cmds))
	}

	for i, cmd := range cmds {
		if cmd.Name != mockCmds[i].Name || cmd.Command != mockCmds[i].Command {
			t.Errorf("Expected command %v, got %v", mockCmds[i], cmd)
		}
	}
}

func TestExecuteCommand_ValidID(t *testing.T) {
	// Arrange
	mockCmds := []entity.Command{
		{Name: "Echo", Command: "echo Hello"},
		{Name: "Sleep", Command: "sleep 1"},
	}

	mockSt := &mockStorage{
		commands: mockCmds,
	}

	mockComm := &mockCommander{}

	cs := NewCommandService(mockSt, mockComm)

	// Act
	execID, err := cs.ExecuteCommand(context.Background(), 0) // Execute "Echo"

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	cs.WaitExecutions(context.Background())

	exec := cs.GetExecution(context.Background(), execID)
	if exec == nil {
		t.Fatalf("Expected execution with ID %d, got nil", execID)
	}

	if exec.ExitCode == nil || *exec.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %v", exec.ExitCode)
	}

	if len(exec.Log) == 0 {
		t.Errorf("Expected log entries, got none")
	}

	found := false
	for _, logEntry := range exec.Log {
		if logEntry.Stream == "stdout" && logEntry.Data == "stdout mock output" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'stdout mock output' in stdout log, got %v", exec.Log)
	}
}

func TestExecuteCommand_InvalidID(t *testing.T) {
	// Arrange
	mockSt := &mockStorage{
		commands: []entity.Command{},
	}

	cs := NewCommandService(mockSt, &mockCommander{})

	// Act
	_, err := cs.ExecuteCommand(context.Background(), 0)

	// Assert
	if err == nil {
		t.Fatalf("Expected error for invalid command ID, got none")
	}

	if !errors.Is(err, CommandNotFoundError) {
		t.Errorf("Expected CommandNotFoundError, got %v", err)
	}
}

func TestExecuteCommand_CommandFailure(t *testing.T) {
	// Arrange
	mockCmds := []entity.Command{
		{Name: "Fail", Command: "exit 1"},
	}

	mockSt := &mockStorage{
		commands: mockCmds,
	}

	mockCmd := &mockCommand{
		runFunc: func() error {
			return errors.New("command failed")
		},
		exitCode: 1,
	}

	mockComm := &mockCommander{
		commandFunc: func(name string, arg ...string) Command {
			return mockCmd
		},
	}

	cs := NewCommandService(mockSt, mockComm)

	// Act
	execID, err := cs.ExecuteCommand(context.Background(), 0)

	// Assert
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	cs.WaitExecutions(context.Background())

	exec := cs.GetExecution(context.Background(), execID)
	if exec == nil {
		t.Fatalf("Expected execution with ID %d, got nil", execID)
	}

	if exec.ExitCode == nil || *exec.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %v", exec.ExitCode)
	}
}

func TestExecuteCommand_OutputCapture(t *testing.T) {
	// Arrange
	mockCmds := []entity.Command{
		{
			Name:    "MixedOutput",
			Command: "echo stdout message && echo stderr message 1>&2",
		},
	}

	mockSt := &mockStorage{
		commands: mockCmds,
	}

	mockCmd := &mockCommand{
		runFunc: func() error {
			return nil
		},
		stdoutPipeFunc: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBufferString("stdout message")), nil
		},
		stderrPipeFunc: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewBufferString("stderr message")), nil
		},
		exitCode: 0,
	}

	mockComm := &mockCommander{
		commandFunc: func(name string, arg ...string) Command {
			return mockCmd
		},
	}

	cs := NewCommandService(mockSt, mockComm)

	// Act
	execID, err := cs.ExecuteCommand(context.Background(), 0)

	// Assert
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	cs.WaitExecutions(context.Background())

	exec := cs.GetExecution(context.Background(), execID)
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
}

func TestGetExecutionHistory(t *testing.T) {
	// Arrange
	mockCmds := []entity.Command{
		{Name: "Echo", Command: "echo Hello"},
	}

	mockSt := &mockStorage{
		commands: mockCmds,
	}

	mockComm := &mockCommander{}

	cs := NewCommandService(mockSt, mockComm)

	_, err := cs.ExecuteCommand(context.Background(), 0)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	cs.WaitExecutions(context.Background())

	// Act
	history, err := cs.GetExecutionHistory(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("GetExecutionHistory failed: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("Expected history length 1, got %d", len(history))
	}

	if history[0].CommandName != "Echo" {
		t.Errorf("Expected CommandName 'Echo', got '%s'", history[0].CommandName)
	}
}

func TestGetExecution(t *testing.T) {
	// Arrange
	mockCmds := []entity.Command{
		{Name: "Echo", Command: "echo Hello"},
	}

	mockSt := &mockStorage{
		commands: mockCmds,
	}

	mockComm := &mockCommander{}

	cs := NewCommandService(mockSt, mockComm)

	execID, err := cs.ExecuteCommand(context.Background(), 0)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	cs.WaitExecutions(context.Background())

	// Act
	exec := cs.GetExecution(context.Background(), execID)

	// Assert
	if exec == nil {
		t.Fatalf("Expected execution with ID %d, got nil", execID)
	}

	if exec.ExecId != execID {
		t.Errorf("Expected ExecId %d, got %d", execID, exec.ExecId)
	}

	if exec.CommandId != 0 {
		t.Errorf("Expected CommandId 0, got %d", exec.CommandId)
	}

	if exec.ExitCode == nil || *exec.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %v", exec.ExitCode)
	}

	if len(exec.Log) == 0 {
		t.Errorf("Expected log entries, got none")
	}
}
