kpsync
======

A small util to launch keepassXC while the database file is on a remote webDAV server (e.g. Nextcloud).

# Usage

1. simply start `kpsync`
2. On first start a example config in `~/.config/kpsync.json` will be created, probably needs to be edited
3. Afterwards start it again
4. You can access the logs, functionality, state etc via the tray icon

# Functionality

kpsync starts by downloading the latest db file from the webDAV to the (configured) temp directory  
(if there already exists a local file, matching with the server version (via ETag), the download will be skipped)

If the download fails, the user gets the option to open a local (fallback) file (e.g. if the computer has no network)

Then KeepassXC is launched.

The temp directory is being watched (inotify) and on file changes they are uploaded to the server.

If there are conflicts (e.g. two clients editing the file at the same time) we ask the user what to do (via `notify-send`)

# Prerequisites

Tested on Linux + Arch + KDE.

Needs `notify-send` to send desktop notifications.  
Needs `inotify` to watch the directory for changes.  
Needs `keepassxc` to be installed. duh.  

# Config (example)

```json
{
    "webdav_url":        "https://cloud.example.com/remote.php/dav/files/YourUser/example.kdbx",
    "webdav_user":       "user",
    "webdav_pass":       "hunter2",
    "local_fallback":    "/home/user/example.kdbx",
    "work_dir":          "/tmp/kpsync",
    "debounce":          3500,
    "terminal_emulator": "konsole -e"
}
```

# Screenshot

<img width="539" height="406" alt="image" src="https://github.com/user-attachments/assets/283ec720-d45d-412a-9be4-b92e9008c9ee" />
