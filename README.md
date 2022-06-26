# go-pst-export

A library and command line utility for exporting PST files (using [go-pst](https://github.com/mooijtech/go-pst)) to various formats (current only supports EML and attachments).

## Usage

### Utility
```bash
# Clone or download go-pst-export
$ git clone https://github.com/mooijtech/go-pst-export

# Change directory
$ cd go-pst-export

# Show help
$ go run cmd/export.go -help

# Export all messages to EML
$ go run cmd/export.go -strategy eml
```
