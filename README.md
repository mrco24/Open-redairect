# install

go install github.com/mrco24/open-redirect@latest

cp -r /root/go/bin/open-redirect /usr/local/bin

# Open-redirect
open-redirect -l url.txt -p payloads.txt -o 2.txt -t 5  -v


