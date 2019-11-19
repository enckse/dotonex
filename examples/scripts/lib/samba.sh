#!/bin/bash
{{ range $key, $user := .Users }}
if pdbedit -L | grep -q "^{{ $user.LoginName }}:"; then
    echo "samba passwd update: {{ $user.LoginName }}"
    printf "%s\n%s\n" {{ $user.Password }} {{ $user.Password }} | smbpasswd -s {{ $user.LoginName }}
fi
{{ end }}
