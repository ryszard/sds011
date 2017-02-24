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

The output will look something like this:

```
pi@raspberrypi ~ $ ./sds011
2017-02-24T11:38:44Z,3.2,3.5
2017-02-24T11:38:46Z,3.2,3.5
2017-02-24T11:38:49Z,3.1,3.4
2017-02-24T11:38:50Z,3.1,3.4
2017-02-24T11:38:53Z,3.2,3.5
2017-02-24T11:38:54Z,3.2,3.6
2017-02-24T11:38:56Z,3.2,3.6
```

This is CSV containing first the timestamp (in RFC3339 format), then
the PM2.5 levels, then the PM10 levels. You can either send it to a
file or pipe it into a script that will export it somewhere (which is
what I am doing, because with the WiFi dongle and sensor connected I
don't have room for a pendrive, so space is scarce).

If you want to do something more fancy, just import
`github.com/ryszard/sds011/go/sds011` in your code.

# License

[Apache 2.0](https://www.tldrlegal.com/l/apache2), please see the file
LICENSE.