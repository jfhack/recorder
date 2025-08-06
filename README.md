# Recorder

A small utility that continuously records RTSP video streams into configurable-length chunks

## Install

The only required dependency is `ffmpeg`

```bash
sudo apt install ffmpeg
```

To install, each release includes an installer, which is `install.sh`

```bash
sudo ./install.sh
```
```
[sudo] password for cantor: 
Enter the user who owns the process
You may use a colon to specify user:group. [root]: cantor:cantor
Enter installation path [/usr/local/bin/]: 
Enter configuration file path [/mnt/ssd/cam/config.yaml]: 
Enter video directory path [/mnt/ssd/cam/videos]: 

Installation details:
Service user: cantor:cantor
Installation binary path: /usr/local/bin/recorder
Configuration file path: /mnt/ssd/cam/config.yaml
Video directory path: /mnt/ssd/cam/videos

This script will install the recorder binary and service.
Proceed with installation? (y/N): y
Warning: Configuration file '/mnt/ssd/cam/config.yaml' does not exist. It will be created during the installation.
Installing new service...
Created symlink /etc/systemd/system/multi-user.target.wants/recorder.service → /etc/systemd/system/recorder.service.
Installation complete.
```

## Configuration

The configuration is a YAML file with camera URLs, a chunk `duration`, and a `slack` value (default 10s). `slack` adds extra recording time to overlap chunks because starting the next RTSP connection may take a moment. Each camera must have a unique `name`. Optional `transport` can be `udp` (default) or `tcp`.

`duration` and `slack` support the units `s`, `m`, and `h`.

Here's an example:
```yaml
cameras:
  - name: p1
    url: rtsp://admin:1234@10.10.1.102:554/onvif1
    duration: 30m
    slack: 10s
  - name: p2
    url: rtsp://admin:1234@10.10.1.105:554/onvif1
    duration: 30m
    slack: 10s
  - name: p3
    url: rtsp://admin:1234@10.10.1.108:554/onvif1
    duration: 30m
    slack: 10s
  - name: p4
    url: rtsp://admin:1234@10.10.1.112:554/onvif1
    duration: 30m
    slack: 10s
```

An example of a flow could be the following with `duration: 30m` and `slack: 10s`:
- p1 chunk 0 starts at 10:55:08
- p1 chunk 1 starts at 11:25:08
- p1 chunk 0 ends at 11:25:18

Note that this indicates the start and end of the process, not necessarily the actual duration of the video chunk. Due to the connection process, depending on the device and connection, it may be advisable to have a `slack: 20s`

## Build

To build you need to have [Go](https://go.dev/doc/install) installed, then simply use the `build.sh` script.

```
└── build
    ├── linux_amd64_v0.1.0.tar.gz
    ├── linux_arm64_v0.1.0.tar.gz
    └── linux_armv7_v0.1.0.tar.gz
```

## Usage

In general, run it as a systemd service: 
```bash
sudo systemctl start recorder
sudo systemctl stop recorder
```

Alternatively, you can run the binary directly:
```bash
./recorder -config config.yaml
```

That way the videos will be created in the current directory in a subdirectory with the name given in the configuration file. In the case of the service, the installation script sets the `WorkingDirectory` in the systemd unit.

An output file structure, from the configuration given above, might look something like this:

```
.
├── config.yaml
├── install.sh
├── recorder
└── videos
    ├── p1
    │   ├── 20250803
    │   │   ├── ...
    │   │   └── p1-20250803-233415.mkv
    │   ├── 20250804
    │   │   ├── ...
    │   │   └── p1-20250804-233415.mkv
    │   └── 20250805
    │       ├── ...
    │       └── p1-20250805-205257.mkv
    ├── p2
    │   ├── 20250803
    │   │   ├── ...
    │   │   └── p2-20250803-233415.mkv
    │   ├── 20250804
    │   │   ├── ...
    │   │   └── p2-20250804-233415.mkv
    │   └── 20250805
    │       ├── ...
    │       └── p2-20250805-205257.mkv
    ├── p3
   ...
```

The utility is intentionally simple and lightweight so it runs efficiently on devices like a Raspberry Pi.
