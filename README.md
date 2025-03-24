# Pull & Deploy Agent

A lightweight and customizable CI/CD agent written in Go.

It automatically pulls code from a GitHub repository, builds the project, and restarts systemd services. Ideal for managing deployments of multiple environments (dev, stage, prod) on a self-managed server (e.g. AWS EC2).

---

## Features

- ✅ Pulls new commits from GitHub on a given interval
- 🔐 Supports private and public GitHub repositories
- 🔧 Rebuilds and restarts systemd services automatically
- 🧩 Multi-environment support (e.g. dev, stage, prod)
- 🔔 Slack notifications for success/failure
- 🪵 Logs to file

---

## Configuration

All settings are managed via a `config.json` file.

### Sample `config.json`

```json
{
  "interval_seconds": 120,
  "log_file": "/home/ubuntu/pull-deploy.log",
  "git_config": {
    "username": "your_github_username",
    "token": "your_personal_access_token",
    "repo_owner": "your_github_org_or_user",
    "repo_name": "your-repo-name",
    "use_auth": true
  },
  "slack": {
    "enabled": true,
    "webhook_url": "https://hooks.slack.com/services/XXX/YYY/ZZZ"
  },
  "environments": [
    {
      "branch": "dev",
      "dir": "/home/ubuntu/your-project-dev",
      "service_name": "your-project-dev"
    },
    {
      "branch": "stage",
      "dir": "/home/ubuntu/your-project-stage",
      "service_name": "your-project-stage"
    },
    {
      "branch": "main",
      "dir": "/home/ubuntu/your-project-prod",
      "service_name": "your-project-prod"
    }
  ]
}
```

> Replace `your-project` with the name of your actual project.

---

## Usage

### Build

```bash
go build -o pull-deploy-agent main.go
```

### Run Manually

```bash
./pull-deploy-agent
```

It will run indefinitely and check for updates every `interval_seconds`.

---

## Run as a systemd Service

### Create systemd unit file:

**/etc/systemd/system/pull-deploy-agent.service**

```ini
[Unit]
Description=Pull & Deploy Agent
After=network.target

[Service]
ExecStart=/home/ubuntu/pull-deploy-agent
WorkingDirectory=/home/ubuntu
Restart=always
User=ubuntu
Environment=GO_ENV=production

[Install]
WantedBy=multi-user.target
```

### Start & Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable pull-deploy-agent
sudo systemctl start pull-deploy-agent
```

### Check Logs:

```bash
journalctl -u pull-deploy-agent -f
```

---

## Slack Notification Samples

- ✅ Success:
  `:rocket: [dev] Deploy successful at Tue, 24 Mar 25 18:33 UTC`

- ❌ Error:
  `:x: [stage] Deploy error: build failed: ...`

---

## Roadmap (Planned)

- [x] Slack notifications
- [ ] Telegram alerts
- [ ] Email alerts
- [ ] Prometheus metrics
- [ ] Web dashboard (optional)

---

## License

MIT License — feel free to use, modify and contribute.