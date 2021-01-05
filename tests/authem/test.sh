_test() {
    {{ range $key, $vlan := .VLANs }}
        echo {{ $vlan }}
    {{ end }}
    {{ range $key, $sys := .Systems }}
        echo {{ $sys }}
    {{ end }}
    {{ range $key, $user := .Users }}
        echo {{ $user.UserName }}
        echo {{ $user.LoginName }}
        echo {{ $user.Password }}
        {{ range $skey, $sys := $user.Systems }}
            echo {{ $sys }}
        {{ end }}
    {{ end }}
}

_test > bin/script.stdout
