# Chat Testing (WIP)

## SCIONLab VM testing
``` shell
cd /git/scion-apps/chat
watcher -a 0.0.0.0 -p 8000 -srvroot ./web -sabin $GOPATH/bin
```

## Localhost creating virtual cameras
### install deps
``` shell
sudo apt install ffmpeg v4l2loopback-dkms v4l-utils
```
### install cams
``` shell
sudo modprobe v4l2loopback devices=2 exclusive_caps=1,1
v4l2-ctl --list-devices
```
### launch cam feeds
``` shell
gnome-terminal --tab --title="video1" -e 'sh -c "ffmpeg -loop 1 -re -i ~/Desktop/virtA.png -f v4l2 -vcodec rawvideo -pix_fmt yuv420p /dev/video1"' &
gnome-terminal --tab --title="video2" -e 'sh -c "ffmpeg -loop 1 -re -i ~/Desktop/virtB.png -f v4l2 -vcodec rawvideo -pix_fmt yuv420p /dev/video2"' &
```

## Localhost testing on test topology
``` shell
gnome-terminal --tab --title="chat 111" --working-directory="/git/scion-apps/chat" -- bash -c "watcher \
-a 127.0.0.1 \
-p 8081 \
-sciond 127.0.0.19:30255 \
-srvroot /git/scion-apps/chat/web \
-sabin $GOPATH/bin; bash" &
gnome-terminal --tab --title="chat 112" --working-directory="/git/scion-apps/chat" -- bash -c "watcher \
-a 127.0.0.1 \
-p 8082 \
-sciond [fd00:f00d:cafe::7f00:b]:30255 \
-srvroot /git/scion-apps/chat/web \
-sabin $GOPATH/bin; bash" &
```
