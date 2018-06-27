URL="facebookcorewwwi.onion"
torsocks curl -s $URL >/dev/null 2>/dev/null
if [ $? -eq 0 ]; then
    echo "[+] $URL is online"
else
    echo "[-] $URL is offline"
fi
