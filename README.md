# nanotools

A self-hostable collection of privacy-first web utilities built in Go.

It's mostly a collection of tools that I often find myself looking up online, but it's usually never clear whether or not my JSON response with API keys, etc. will end up in the wrong hands.

## stack

- Go
- Templ
- chi
- SQLite
- sqlc

## Features

Privacy First: All processing happens on your server
✅ JSON Formatter: Validate and beautify JSON
[] Text Utils: Base64, slugify, UUID, hashing
[] Image Tools: Convert, compress, and process images
[] PDF Tools: Convert PDFs to images
[] Video Tools: Create GIFs from video clips
[] Easy deploy with Docker and kamal