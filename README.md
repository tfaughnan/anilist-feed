# anilist-feed

Generates an [Atom feed](https://www.rfc-editor.org/rfc/rfc4287.html) for
user activity on [AniList](https://anilist.co) and prints to stdout by default.
Run it in a cronjob or systemd timer and serve the file with a static web
server. E.g.

```sh
anilist-feed -n $username \
        -o /srv/http/mirror.tjfjr.net/feeds/anilist/$username.xml \
        -u https://mirror.tjfjr.net/feeds/anilist/$username.xml \
        -e tjfjr.net,2024
```
