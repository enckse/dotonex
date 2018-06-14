BIN=bin/
AUDITS=${BIN}audit.csv
if [ ! -e $AUDITS ]; then
    exit 0
fi

source /etc/environment
if [ -z "$RPT_HOST" ]; then
    echo "missing RPT_HOST var"
    exit 1
fi

if [ -z "$RPT_TOKEN" ]; then
    echo "missing RPT_TOKEN var"
    exit 1
fi

_post() {
    for f in $(ls $BIN | grep "\.md"); do
        content=$(cat $BIN/$f | python -c "import sys, urllib.parse; print(urllib.parse.quote(sys.stdin.read()))")
        name=$(echo "$f" | cut -d "." -f 1)
        curl -s -k -X POST -d "name=$name&content=$content" "$RPT_HOST/reports/upload?session=$RPT_TOKEN"
    done
}

DAILY=1
if [ ! -z "$1" ]; then
    DAILY=$1
fi

# User.VLAN macs assigned
ASSIGNED=${BIN}assigned.md
echo "| vlan | user | mac |
| --- | --- | --- |" > $ASSIGNED

cat $AUDITS | sed "s/,/ /g" | awk '{print "| " $2 " | " $1 " | " $3 " |"}' | sort -u >> $ASSIGNED

if [ $DAILY -ne 1 ]; then
    _post
    exit 0
fi

# Auth information
AUTHS=${BIN}auths.md
echo "| user | mac | last |
| --- | --- | --- |" > $AUTHS

dates=$(date +%Y-%m-%d)
for i in $(seq 1 10); do
    dates="$dates "$(date -d "$i days ago" +%Y-%m-%d)
done
files=""
for d in $(echo "$dates"); do
	f="/var/lib/radiucal/log/radiucal.proxy.audit.$d"
	if [ -e $f ]; then
		files="$files $f"
	fi
done
if [ ! -z "$files" ]; then
    notcruft=""
    users=$(cat $files \
            | cut -d " " -f 3,4 \
            | sed "s/ /,/g" | sort -u)
    for u in $(echo "$users"); do
        for f in $(echo "$files"); do
            has=$(tac $f | sed "s/ /,/g" | grep "$u" | head -n 1)
            if [ ! -z "$has" ]; then
                day=$(basename $f | cut -d "." -f 3)
                stat=$(echo $has | cut -d "," -f 2 | sed "s/\[//g;s/\]//g")
                usr=$(echo $u | cut -d "," -f 1)
                notcruft="$notcruft|$usr"
                mac=$(echo $u | cut -d "," -f 2 | sed "s/(//g;s/)//g;s/mac://g")
                echo "| $usr | $mac | $stat ($day) |" >> $AUTHS
                break
            fi
        done
    done
    notcruft=$(echo "$notcruft" | sed "s/^|//g")
    cat $AUDITS | sed "s/,/ /g" | awk '{print $2,".",$1}' | sed "s/ //g" | grep -v -E "($notcruft)" | sed "s/^/drop: /g" | sort -u | smirc --report
fi

# Leases
LEASES_KNOWN=${BIN}known_leases
rm -f $LEASES_KNOWN
LEASES=${BIN}leases.md
echo "| status | mac | ip |
| --- | --- | --- |" > $LEASES
unknowns=""
leases=$(curl -s -k "$RPT_HOST/reports/view/dns?raw=true")
for l in $(echo "$leases" | sed "s/ /,/g"); do
    t=$(echo $l | cut -d "," -f 1)
    ip=$(echo $l | cut -d "," -f 3)
    mac=$(echo $l | cut -d "," -f 2 | tr '[:upper:]' '[:lower:]' | sed "s/://g")
    line="| $mac | $ip |"
    if [[ "$t" == "static" ]]; then
        echo "| mapped $line" >> $LEASES_KNOWN
        continue
    fi
    if [ ! -z "$LEASE_MGMT" ]; then
        echo "$ip" | grep -q "$LEASE_MGMT"
        if [ $? -eq 0 ]; then
            echo "| mgmt $line" >> $LEASES_KNOWN
            continue
        fi
    fi
    if [ ! -z "$LEASE_MACVLAN" ]; then
        macvlan=$(echo $l | cut -d "," -f 5)
        if [ ! -z "$macvlan" ]; then
            matched=0
            for m in $(echo "$LEASE_MACVLAN"); do
                if echo $macvlan | grep -q "$m"; then
                    echo "| macvlan $line" >> $LEASES_KNOWN
                    matched=1
                    break
                fi
            done
            if [ $matched -eq 1 ]; then
                continue
            fi
        fi
    fi
    cat $AUDITS | grep -q "$mac"
    if [ $? -eq 0 ]; then
        echo "| dhcp $line" >> $LEASES_KNOWN
        continue
    fi
    unknowns="$unknowns $mac ($ip)"
    echo "| unknown $line" >> $LEASES
done
if [ ! -z "$unknowns" ]; then
    echo "unknown leases: $unknowns" | smirc
fi
cat $LEASES_KNOWN | sort -u >> $LEASES

_post
