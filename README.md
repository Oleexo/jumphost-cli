# 🚀 jumphost CLI

> **Your friendly neighborhood AWS RDS tunnel wizard** ✨

Stop fumbling with bash scripts and manual port forwarding! This sleek Go-powered CLI brings you a beautiful 
interactive TUI experience for connecting to your AWS RDS instances through EC2 jumphosts. Built with 
[Bubble Tea](https://github.com/charmbracelet/bubbletea) for that sweet terminal eye candy. 🍵

## ✨ What Makes This Awesome

🎨 **Beautiful Interactive TUI** - Select databases and jumphosts with a gorgeous terminal UI  
🔐 **AWS SSO Support** - Seamlessly works with your SSO profiles  
🎯 **Smart Defaults** - Sensible choices that just work  
🏠 **Auto /etc/hosts** - Optional hostname mapping that cleans up after itself  
🤖 **JSON Mode** - Perfect for scripting and automation  
📡 **Port Forwarding** - Secure SSM tunnels via AWS Session Manager  
🎭 **Flexible Modes** - Interactive, non-interactive, basic, or JSON output  
🧪 **Dry Run** - Preview what will happen before committing

## 🎬 See It In Action

### Interactive Flow - Database Selection

When you run `jumphost connect -i`, you're greeted with a sleek interface:

```
┌─────────────────────────────────────────────────────────────────────┐
│ Acct 123456789012 User admin@company.com - Select RDS Endpoint     │
├─────────────────────────────────────────────────────────────────────┤
│ > prod-postgres-db.abc123.us-east-1.rds.amazonaws.com:5432         │
│   Production PostgreSQL Database                                    │
│                                                                      │
│   staging-mysql-db.xyz789.us-west-2.rds.amazonaws.com:3306         │
│   Staging MySQL Database                                            │
│                                                                      │
│   dev-postgres-db.def456.eu-west-1.rds.amazonaws.com:5432          │
│   Development PostgreSQL Database                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
↑/k up • ↓/j down • / filter • enter select • q quit
```

### Local Port Configuration

```
┌─────────────────────────────────────────────────────────────────────┐
│ Selected: prod-postgres-db.abc123.us-east-1.rds.amazonaws.com:5432 │
│                                                                      │
│ Enter local port (default: 5432):                                   │
│ > 15432_                                                             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Jumphost Selection (when multiple instances found)

```
┌─────────────────────────────────────────────────────────────────────┐
│ Select Jumphost Instance                                            │
├─────────────────────────────────────────────────────────────────────┤
│ > i-0123456789abcdef0                                               │
│   EC2 jumphost instance                                             │
│                                                                      │
│   i-0fedcba9876543210                                               │
│   EC2 jumphost instance                                             │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
↑/k up • ↓/j down • enter select • q quit
```

### Connection Established! 🎉

```
✓ /etc/hosts updated: prod-postgres-db.abc123.us-east-1.rds.amazonaws.com → 127.0.0.1

🚀 Port forwarding session started!
   Instance:   i-0123456789abcdef0
   Endpoint:   prod-postgres-db.abc123.us-east-1.rds.amazonaws.com
   Remote:     5432
   Local:      15432

💡 Connect with: psql -h localhost -p 15432 -U your_user

Press Ctrl+C to stop...
```

## 🛠️ Install

### Go Install (easiest)

```bash
go install github.com/Oleexo/jumphost-cli@latest
```

### From Release

```bash
# Download the latest release for your platform
tar -xzf jumphost_<version>_<os>_<arch>.tar.gz -C /usr/local/bin jumphost
```

## 🎮 Usage

### 🌟 Quick Start - Interactive Mode

The most fun way to use jumphost:

```bash
jumphost connect -i
```

With AWS SSO profile:

```bash
jumphost connect -i --profile my-sso
```

### ⚡ Direct Mode (Non-Interactive)

When you know exactly what you want:

```bash
jumphost connect --endpoint db.example.amazonaws.com:5432 --local-port 15432
```

With AWS profile:

```bash
jumphost connect --endpoint db.example.amazonaws.com:5432 --profile my-sso
```

### 🔍 Dry Run Mode

Preview what will happen without actually doing it:

```bash
jumphost connect --endpoint db.example.amazonaws.com:5432 --dry-run
```

### 🎯 Advanced Options

**Skip /etc/hosts modification:**

```bash
jumphost connect --endpoint db.example.amazonaws.com:5432 --no-hosts
```

**Explicit jumphost instance (skip selection):**

```bash
jumphost connect --endpoint db.example.amazonaws.com:5432 --jumphost-instance i-0123456789abcdef0
```

**Custom jumphost tag:**

```bash
jumphost connect --jumphost-tag myjump
```

## 🎭 Different Modes for Different Needs

### 📺 Basic Mode (No TTY)

Perfect for IDEs, CI/CD, or environments without full terminal support:

```bash
jumphost connect -i --basic
```

Or set the environment variable:

```bash
export JUMPHOST_BASIC=1
jumphost connect -i
```

Basic mode gives you simple numbered lists:

```
Available RDS endpoints:
1) prod-postgres-db.abc123.us-east-1.rds.amazonaws.com:5432
2) staging-mysql-db.xyz789.us-west-2.rds.amazonaws.com:3306
3) dev-postgres-db.def456.eu-west-1.rds.amazonaws.com:5432

Select endpoint (1-3, or q to quit): 
```

### 🤖 JSON Mode (For Scripting)

Machine-readable output for automation:

```bash
jumphost connect --endpoint db.example.amazonaws.com:5432 --json --dry-run
```

Sample output:

```json
{"type":"config_loaded","region":"us-east-1","profile":""}
{"type":"selection","endpoint":"db.example.amazonaws.com","remotePort":5432,"localPort":5432,"applyHosts":true}
{"type":"instance_selected","instanceID":"i-0123456789abcdef0"}
{"type":"dry_run","instanceID":"i-0123456789abcdef0","endpoint":"db.example.amazonaws.com","remotePort":5432,"localPort":5432,"region":"us-east-1","profile":"","applyHosts":true}
```

**Event types emitted:**
- `config_loaded` - AWS config initialized
- `selection` - User made their choices
- `instance_selected` - Jumphost chosen
- `hosts_applied` - /etc/hosts updated
- `hosts_error` - Hosts update failed
- `dry_run` - Dry run summary
- `session_start` - Port forwarding active
- `session_end` - Session terminated
- `cancelled` - User cancelled
- `error` - Something went wrong

## 🎛️ Environment Variables

Fine-tune behavior with these environment variables:

| Variable                 | Effect                            |
| ------------------------ | --------------------------------- |
| `JUMPHOST_BASIC=1`       | Force basic mode (no TUI)         |
| `JUMPHOST_FORCE_PTY=1`   | Force pseudo-terminal for AWS CLI |
| `JUMPHOST_DISABLE_PTY=1` | Disable PTY/script fallback       |
| `JUMPHOST_DEBUG=1`       | Verbose diagnostics               |

### 🔧 Controlling PTY Behavior

The AWS Session Manager plugin can be picky about TTYs. We've got you covered:

1. **Auto-magic mode** (default): Try direct exec → fall back to PTY → fall back to `script`
2. **Force PTY**: `export JUMPHOST_FORCE_PTY=1`
3. **Disable PTY**: `jumphost connect ... --no-pty`
4. **Debug**: `export JUMPHOST_DEBUG=1` to see what's happening

## 🔐 AWS SSO & Profile Support

The `--profile` flag makes AWS SSO a breeze:

```bash
# First, login to your SSO
aws sso login --profile my-sso

# Then use jumphost with that profile
jumphost connect -i --profile my-sso
```

The profile is used for both AWS SDK operations and the underlying `aws ssm start-session` command.

## ✅ Requirements

- 🔧 AWS CLI installed and configured
- 🔑 AWS credentials with SSM, RDS Describe, and EC2 Describe permissions
- 🎯 EC2 jumphost instances tagged (default: `Usage=jumphost`)
- 📡 SSM Agent running on jumphost instances

## 🧪 Development

### Run Tests

```bash
go test ./...
```

### Build Locally

```bash
make build
```

### Release Workflow

Version metadata is automatically injected via ldflags:

```bash
# Show current version
make version

# Build snapshot (local testing)
make snapshot

# Create release (requires tag)
git tag v0.3.0
git push origin v0.3.0
GITHUB_TOKEN=xxxx make release
```

### 🤖 CI/CD

**Automated Testing**: GitHub Actions runs tests on every push/PR

**Automated Releases**: Push a tag like `v0.3.0` and GitHub Actions will:
- Build for multiple platforms
- Create a GitHub Release
- Upload binaries and checksums

**Manual Snapshots**: Trigger workflow manually with `snapshot=true` for testing

## 📚 How It Works

1. 🔍 **Discover**: Lists your RDS instances using AWS API
2. 🎯 **Select**: Choose your database (and jumphost if needed)
3. 🔌 **Connect**: Uses AWS SSM `StartPortForwardingSessionToRemoteHost` document
4. 🏠 **Map** (optional): Temporarily adds hostname to /etc/hosts
5. 🧹 **Cleanup**: Restores everything on exit (Ctrl+C safe!)

## 🎨 Tech Stack

- 🦫 **[Cobra](https://github.com/spf13/cobra)** - CLI framework
- 🍵 **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** - TUI framework
- 🫧 **[Bubbles](https://github.com/charmbracelet/bubbles)** - TUI components
- 💄 **[Lipgloss](https://github.com/charmbracelet/lipgloss)** - Terminal styling
- ☁️ **[AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)** - AWS integration
- 🚀 **[GoReleaser](https://goreleaser.com)** - Release automation

## 🤝 Contributing

Found a bug? Have an idea? PRs welcome! This tool shells out to `aws ssm start-session` rather than 
reimplementing the protocol, keeping things maintainable and compatible.

## 📝 License

Check the repository for license information.

---

<div align="center">

**Made with ❤️ and ☕**

*Stop wrestling with bash, start tunneling in style* ✨

</div>

