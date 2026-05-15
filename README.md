# Subscreen
ffmpeg wrapper that takes screenshots at subtitle timecodes and outputs a JSON file mapping each subtitle to its screenshots.

I made this so I don't need to manually take screenshots from the awful 2014 movie Seventh Son which is an adaption of the novel The Spook's Apprentice by Joseph Delaney

Should in theory work on every OS but I have not tested outside of Windows

## Build

```
.\build.ps1
```

Runs `go vet`, tests, and builds to `dist/subscreen.exe`.

## Usage

```
subscreen -video <file> [flags]
```

If `-srt` is not given, it looks for a `.srt` file next to the video and prompts before using it

Reccomend using `-one-per-subtlte` and `-try-sync` nearly every time, as otherwise you will have a large number of screenshots that may be out of sync

## Flags

| Flag | Default | Description |
|---|---|---|
| `-video` | | video file path (required) |
| `-srt` | | SRT subtitle file |
| `-format` | `jpeg` | output image format: `jpeg` or `png` |
| `-out-dir` | `screenshots` | directory to write screenshots into |
| `-out-json` | `output.json` | path for the output JSON file |
| `-start` | | only process subtitles after this time (e.g. `30m`) |
| `-end` | | only process subtitles before this time (e.g. `35m`) |
| `-delay` | `1s` | interval between screenshots within a subtitle |
| `-one-per-subtitle` / `-ops` | | take one screenshot per subtitle (at the midpoint) |
| `-offset` / `-o` | `0` | shift all subtitle timestamps (e.g. `2s`, `-500ms`) |
| `-try-sync` / `-ts` | `1.0` | rate multiplier for linear drift correction; `adjusted = srt_time x rate + offset` |
| `-fast-seek` / `-fs` | | fast but less accurate seeking (may capture wrong frame) |
| `-y` | | answer yes to all prompts |

## Output

```json
{
  "video": "video.mp4",
  "subtitles": "video.srt",
  "entries": [
    {
      "index": 1,
      "start": "00:00:01,000",
      "end": "00:00:04,000",
      "text": "Hello world",
      "screenshots": ["screenshots/0001_00-00-01-000.jpg"]
    }
  ]
}
```

## Requirements

- Go 1.26+
- ffmpeg on PATH

## Known issues
- Takes screenshots even when it is a quiet moment where no subtitles are shown
- i have watched Seventh Son (2014) over 30 times, please help me
