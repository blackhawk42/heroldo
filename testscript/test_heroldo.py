import argparse
import mimetypes
import os
import sys
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
from uuid import uuid4


def build_multipart(fields, multi_fields, files, no_content_type):
    boundary = uuid4().hex
    body = bytearray()

    for name, value in fields.items():
        body.extend(f"--{boundary}\r\n".encode())
        body.extend(f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode())
        body.extend(f"{value}\r\n".encode())

    for name, value in multi_fields:
        body.extend(f"--{boundary}\r\n".encode())
        body.extend(f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode())
        body.extend(f"{value}\r\n".encode())

    for field_name, filepath in files:
        with open(filepath, "rb") as f:
            data = f.read()
        filename = os.path.basename(filepath)
        body.extend(f"--{boundary}\r\n".encode())
        body.extend(
            f'Content-Disposition: form-data; name="{field_name}"; filename="{filename}"\r\n'.encode()
        )
        if not no_content_type:
            ctype, _ = mimetypes.guess_type(filepath)
            if ctype is None:
                ctype = "application/octet-stream"
            body.extend(f"Content-Type: {ctype}\r\n".encode())
        body.extend(b"\r\n")
        body.extend(data)
        body.extend(b"\r\n")

    body.extend(f"--{boundary}--\r\n".encode())
    return bytes(body), boundary


def main():
    parser = argparse.ArgumentParser(
        description="Send multipart form data to heroldo HTTP server"
    )
    parser.add_argument("url", help="Server URL (e.g. http://localhost:8080)")
    parser.add_argument(
        "files", nargs="*", metavar="FILE", help="File(s) to upload (optional)"
    )
    parser.add_argument(
        "-t", "--text", help="Optional message text accompanying the files"
    )
    parser.add_argument(
        "-c",
        "--channel",
        dest="channels",
        action="append",
        metavar="CHANNEL_ID",
        help="Discord channel ID (repeatable, overrides server default)",
    )
    parser.add_argument(
        "--spoiler",
        action="store_true",
        help="Mark all uploaded files as spoilers",
    )
    parser.add_argument(
        "--no-content-type",
        action="store_true",
        help="Omit Content-Type from file parts (triggers server-side autodetection)",
    )
    parser.add_argument(
        "-a",
        "--authorization",
        help="Value for the Authorization header (e.g. 'Bearer <token>')",
    )

    args = parser.parse_args()

    for fp in args.files:
        if not os.path.isfile(fp):
            print(f"error: file not found: {fp}", file=sys.stderr)
            sys.exit(1)

    fields = {}
    if args.text is not None:
        fields["text"] = args.text

    multi_fields = []
    if args.channels:
        for ch in args.channels:
            multi_fields.append(("channels", ch))

    if args.files:
        spoiler_val = "true" if args.spoiler else "false"
        multi_fields.extend(("spoilers", spoiler_val) for _ in args.files)
        files = [("files", fp) for fp in args.files]
    else:
        files = []

    body, boundary = build_multipart(fields, multi_fields, files, args.no_content_type)

    req = Request(
        args.url,
        data=body,
        headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
        method="POST",
    )
    if args.authorization is not None:
        req.add_header("Authorization", args.authorization)

    try:
        with urlopen(req) as resp:
            print(f"Status: {resp.status} {resp.reason}")
            print(f"Response: {resp.read().decode()}")
    except HTTPError as e:
        print(f"Status: {e.code} {e.reason}", file=sys.stderr)
        print(f"Response: {e.read().decode()}", file=sys.stderr)
        sys.exit(1)
    except URLError as e:
        print(f"error: {e.reason}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
