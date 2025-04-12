# Installation and Setup Guide

Follow the steps below to set up the environment and run the application.

## Prerequisites

Ensure you have the following installed on your system:
- A Linux-based operating system
- `sudo` privileges

## Step 1: Install Necessary Dependencies

Run the following commands to install the required dependencies:

```bash
sudo apt-get update
sudo apt-get install -y --no-install-recommends git pkg-config libcap-dev libsystemd-dev ca-certificates make gcc g++ cmake python3 python3-pip python3-venv ninja-build libgtest-dev valgrind
```

## Step 2: Clone and Install Isolate

Clone the `isolate` repository and install it:

```bash
git clone https://github.com/ioi/isolate.git /isolate
cd /isolate
sudo make install
rm -rf /isolate
```

## Step 3: Set Up Sandbox Directories

Create and configure the necessary sandbox directories:

```bash
sudo mkdir -p /sandbox /sandbox/code /sandbox/repo
sudo chmod 777 /sandbox /sandbox/code /sandbox/repo
```

## Step 4: Modify `main.go` for Custom Judging

To customize the judging process, modify the `main.go` file. Update the `sandbox.SandboxPtr.RunShellCommand` function to specify how the judging should be performed. For example:

```go
sandbox.SandboxPtr.RunShellCommand([]byte("/usr/bin/cat text.txt"), []byte(codePath))
```

This command will execute `/usr/bin/cat text.txt` within the sandbox environment. Replace this command with the desired logic for your specific judging requirements.

## Step 5: Run the Application

Navigate to the project directory and run the application:

```bash
go run main.go
```

## Notes

- Ensure all dependencies are installed correctly before running the application.
- The sandbox directories are configured with open permissions (`777`) for demonstration purposes. Adjust permissions as needed for your use case.
- Modifying `main.go` allows you to tailor the judging process to your application's needs.
