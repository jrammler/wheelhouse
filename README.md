# Wheelhouse

Wheelhouse is a web-based tool to execute commands on a server. It provides a user interface to trigger predefined commands and view their execution history and logs.

> [!CAUTION]
> This tool allows users to trigger execution of commands over the network.
> Use at your own risk.
>
> This is just a hobby project and is not ready for production.
> There might be serious security issues.
> 
> Do not expose the server to the internet or run it as root.

## Features

-   **Command Execution**: Execute predefined commands through a web interface.
-   **Execution History**: View the history of command executions, including status, execution time, and logs.
-   **User Authentication**: Secure access with user authentication.
-   **Role-Based Access Control**: Limit command execution based on user roles.

## Getting Started

### Prerequisites

- Go (version 1.22 or higher)

### Installation

Install the application using the following command:

```bash
go install github.com/jrammler/wheelhouse/cmd/wheelhouse@latest
```

This command will download and compile the application and place the executable in your `$GOPATH/bin` directory. Ensure that `$GOPATH/bin` is in your system's `PATH` environment variable.

### Usage

#### Starting the server

You can run the server using the `serve` subcommand:

```bash
wheelhouse serve <addr> <config-file>
```

For example:

```bash
wheelhouse serve :8080 ~/.config/wheelhouse/config.json
```

This will start a server on port `8080` and read the configuration from the file `~/.config/wheelhouse/config.json`

#### Starting the server

To generate a password hash for the config-file, you can use the `hash-password` subcommand:

```bash
wheelhouse hash-password
```

This will ask you to input a password on the terminal and print the hash.

### Configuration

The application uses a JSON configuration file to define commands and users. The configuration file should contain a JSON object with two keys: `commands` and `users`.

#### Commands

The `commands` key should contain a JSON array of command objects. Each command object should have the following keys:

-   `name`: A string representing the name of the command.
-   `command`: A string representing the command to execute.
-   `role` (optional): A string representing the role required to execute the command. If this is omitted, no role is required.

Example:

```json
{
    "name": "say hello",
    "command": "echo \"Hello World\"",
    "role": "admin"
}
```

#### Users

The `users` key should contain a JSON array of user objects. Each user object should have the following keys:

-   `username`: A string representing the username.
-   `password_hash`: A string representing the bcrypt hash of the user's password. You can generate this hash using the `wheelhouse hash-password` command.
-   `roles` (optional): A JSON array of strings representing the roles assigned to the user.

Example:

```json
{
    "username": "user1",
    "password_hash": "$2a$12$jWES6ld8Szh1k3vrgiy.RO2WcVlepnvzZk3KpwJ7OY44lKpO4beg2",
    "roles": ["admin"]
}
```

#### Complete Example

```json
{
    "commands": [
        { "name": "say hello", "command": "echo \"Hello World\"" },
        { "name": "sleep 2 seconds", "command": "sleep 2" }
    ],
    "users": [
        { "username": "admin", "password_hash": "$2a$12$jWES6ld8Szh1k3vrgiy.RO2WcVlepnvzZk3KpwJ7OY44lKpO4beg2" }
    ]
}
```
