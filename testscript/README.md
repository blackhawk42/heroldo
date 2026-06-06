# testscript

Test images and Python test script for the heroldo Discord bot HTTP endpoint.

## Usage

```sh
python testscript/test_heroldo.py <url> [file ...] [options]
```

### Options

| Flag | Description |
|---|---|
| `-t, --text TEXT` | Optional message text accompanying the files |
| `-c, --channel CHANNEL_ID` | Discord channel ID (repeatable, overrides server default) |
| `--spoiler` | Mark all uploaded files as spoilers |
| `--no-content-type` | Omit Content-Type from file parts (triggers server-side autodetection) |
| `-a, --authorization VALUE` | Value for the Authorization header (e.g. `Bearer <token>`) |

### Examples

Upload a single image:

```sh
python testscript/test_heroldo.py http://localhost:8080 testscript/cat.jpg
```

Upload multiple files with text and spoilers, omitting Content-Type:

```sh
python testscript/test_heroldo.py http://localhost:8080 \
  testscript/cat.jpg \
  testscript/dog.jpg \
  --text "look at these animals" \
  --spoiler \
  --no-content-type
```

Upload with a token authorization header:

```sh
python testscript/test_heroldo.py http://localhost:8080 image.png -a "Bearer abc123"
```

Send text-only with specific channels:

```sh
python testscript/test_heroldo.py http://localhost:8080 \
  --text "hello" \
  --channel 1037058660028919831 \
  --channel 1037058660028919832
```
