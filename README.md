# Recorder

A small utility that continuously records FFmpeg-compatible live video streams into video chunks

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

| Field | Type / Format | Units / Values | Description |
| - | - | - | - |
| `url` | string (URL) || URLs of the cameras. |
| `duration` | string | `s`, `m`, `h` | Duration of the process that records the video, not necessarily the video itself, but it gives an approximation. |
| `slack` | string | `s`, `m`, `h` (default `10s`) | Extra recording time to overlap chunks, compensates for connection startup delay. |
| `name` | string || Unique name for each camera. |
| `autoArgs` | boolean | `true` / `false` (default `true`) | If enabled, arguments will be updated automatically for better performance. |
| `args` | dictionary || Contains argument lists for FFmpeg: |
| - `global` | list || Arguments applied globally. |
| - `input` | list || Arguments applied before the input URL. |
| - `output` | list || Arguments applied before the output file path. |
| `suffix` | string | default `.mkv` | The output file's suffix or extension. Includes the dot. *See note below* |


Here some examples:

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

```yaml
cameras:
  - name: park
    url: http://172.19.20.228/cam/live.m3u8
    duration: 1m
    slack: 20s
    args:
      global: ["-hide_banner", "-loglevel", "error"]
      output:
        - -c:v
        - h264_nvenc
        - -profile:v
        - high
        - -bf
        - 2
        - -g
        - 30
        - -crf
        - 18
        - -pix_fmt
        - yuv420p
  - name: street
    url: http://172.19.20.229/cam/live.m3u8
    duration: 10m
```

This is what the final command would look like:

```bash
ffmpeg [global...] [input...] -i url [output...] name/yyyymmdd/name-yyyymmdd-hhmmss[suffix]
```

> [!IMPORTANT]
> It's recommended to keep the default `.mkv` `suffix`.
> When recording/transcoding live streams with FFmpeg, MKV is safer and more flexible than MP4. MKV remains playable even if the process or stream is interrupted (MP4 often doesn’t, due to its end-of-file index)

> [!NOTE]
> It's important to note that if the source is an HLS stream, you need to make sure it's a **live** stream. Otherwise, the recording will always start from the same point, likely increasing in length over time.

An example of a flow could be the following with `duration: 30m` and `slack: 10s`:
- p1 chunk 0 starts at 10:55:08
- p1 chunk 1 starts at 11:25:08
- p1 chunk 0 ends at 11:25:18

Note that this indicates the start and end of the process, not necessarily the actual duration of the video chunk. Due to the connection process, depending on the device and connection, it may be advisable to have a `slack: 20s`

### autoArgs

* If `autoArgs` is `true` (the default) and the `url` starts with `rtsp`, then the input arguments list will automatically have `-rtsp_transport` `udp` appended.
* If `autoArgs` is `true` and the output arguments don't specify any codec options or include options incompatible with stream copy, then `-c` `copy` will be added.


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

An output file structure, from the first configuration given above, might look something like this:

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
