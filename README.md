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

## License

go-pst-export is open-source under the GNU Affero General Public License Version 3 - AGPLv3. Fundamentally, this means that you are free to use go-pst for your project, as long as you don't modify go-pst. If you do, you have to make the modifications public.

## Contact

Feel free to contact me if you have any questions.
Name: Marten Mooij
Email: info@mooijtech.com