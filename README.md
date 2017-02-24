Nova SDS011 PM sensor serial reader written in Go.

# Quickstart

I am assuming that your SDS011 is connected to a Raspberry Pi, like
mine. In my experience it's easier to cross compile the reader on your
laptop and then scp it to the Pi. The fact that Go compiles everything
statically means you don't have to worry about dependencies, which is
nice.

```
$ GOOS=linux GOARCH=arm go build  ./go/cmd/sds011 && scp  sds011 pi@pi:
```

Depending on the version of your Pi, you may need to add `GOARM=6`.

The binary will output something like this:

```
2017-02-24T11:01:51Z	3.4	3.8
2017-02-24T11:01:54Z	3.4	3.8
2017-02-24T11:01:55Z	3.4	3.8
2017-02-24T11:01:56Z	3.4	3.8
2017-02-24T11:01:57Z	3.4	3.8
2017-02-24T11:01:58Z	3.4	3.9
2017-02-24T11:01:59Z	3.5	4
2017-02-24T11:02:00Z	3.6	4.1
2017-02-24T11:02:01Z	3.6	4.2
```

This is TSV contianing first the timestamp (in RFC3339 format), then
the PM2.5 levels, then the PM10 levels. You can either send it to a
file or pipe it into a script that will export it somewhere (which is
what I am doing, because with the WiFi dongle and sensor connected I
don't have room for a pendrive, so space is scarce).

# License

[Apache 2.0](https://www.tldrlegal.com/l/apache2), please see the file
LICENSE.