templ generate
GOOS=linux GOARCH=386 go build -o tibiabuddy
ssh linode 'service tibiabuddy stop'
scp tibiabuddy linode:/root/apps/
ssh linode 'service tibiabuddy start'

