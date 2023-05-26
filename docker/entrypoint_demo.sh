#!/bin/bash

log(){
    echo "[+] $1"
}

WEBROOT="/webroot"
DEMO_DATA="$WEBROOT/demo-data"

if [[ $RANDOMIZE_WEBROOT != "" ]]; then
    echo "Randomizing webroot"
    if [[ ! -d "$DEMO_DATA" ]]; then
        mkdir -p "$DEMO_DATA"
    fi
    chunks=(1 10 100)
    cd "$DEMO_DATA"
    for v in ${chunks[@]} ; do
        outfile="${v}M.data"
        log "Generating $outfile file..."
        dd if=/dev/random of="$outfile" bs=1M count=$v
        final="$(md5sum $outfile|sed -r -e 's/\s.+//')-$outfile"
        mv "$outfile" "$final"
    done
    cd -
fi

log "Generating certificates"

skyhook x509 generate-config > x509-config.yml && \
    skyhook x509 generate-certs && \
    rm x509-config.yml

log "Generating config file"

skyhook server generate-config -r 10 | \
    sed -r -e 's/cert_path.+/cert_path: cert.pem/' | \
    sed -r -e 's/key_path.+/key_path: key.pem/' | \
    sed -r -e 's/interface: lo/interface: eth0/' | \
    sed -r -e "s/your\.fqdn\.here/$LINK_SOCKET/g" > config.yml

log "Dumping latest user accounts:"

grep -A8 users config.yml

log "^^^^^^ NOTE: Log in to the admin interface for convenient access to ^^^^^^"

log "Starting Skyhook"
skyhook server run -c config.yml
