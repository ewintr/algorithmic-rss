# Algorithmic RSS

Algorithmic RSS is an attempt to create a personal algorithmic feed of new using RSS, Miniflux and local LLMs.

## Install as systemd service

Create user and group:

```bash
$ sudo adduser --system --no-create-home --home /nonexistent --shell /sbin/nologin algorithmic-rss
$ sudo groupadd algorithmic-rss
$ sudo usermod -aG algorithmic-rss algorithmic-rss
```

Save unit file to `/etc/systemd/system/algorithmic-rss.service`:

```
[Unit]
Description=Algorithmic RSS service
After=network-online.target

[Service]
ExecStart=/usr/local/bin/algorithmic-rss
User=algorithmic-rss
Group=algorithmic-rss
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
```

Make sure the binary is copied to the right location: `/usr/local/bin/algorithmic-rss`

Enable service:

```bash
$ sudo systemctl daemon-reload
$ sudo systemctl enable algorithmic-rss
$ sudo systemctl start algorithmic-rss
```

Check with:

```bash
$ sudo journalctl -f -u algorithmic-rss
```


