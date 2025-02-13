# lfi2rce_phpinfo.go

This program is used to exploit a Remote Code Execution vulnerability by uploading a file to the server, retrieving it's random name by using phpinfo and executing it through a Local File Inclusion (LFI) vulnerability.

## Usage

```bash
go run lfi2rce_phpinfo.go -phpinfo http://10.10.176.70/dashboard/phpinfo.php -lfi http://10.10.176.70/dev/index.html?view=%s
```

## To-Do:
- [ ] TLS/SSL support
