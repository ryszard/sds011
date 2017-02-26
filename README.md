Nova SDS011 PM sensor serial reader written in Go.

# Quickstart

I am assuming that your SDS011 is connected to a Raspberry Pi, like
mine. In my experience it's easier to cross compile the reader on your
laptop and then scp it to the Pi. The fact that Go compiles everything
statically means you don't have to worry about dependencies, which is
nice.

```
$ GOOS=linux GOARCH=arm go build  ./go/cmd/sds011 && rsync --progress -v -e ssh sds011 pi@pi:
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
the PM2.5 levels, then the PM10 levels.

# Usage

As the output of `sds011` is CSV, it should be easy to process. There
are a few things you can do. First, you can save to a file and either
process it right there, or export it for processing. Second, you can
pipe it into a script that will export it. This is what I am doing, as
disk space is scarce on my Pi (one USB port is used by the meter, the
other by the WiFi dongle, so I there's no other storage connected).


If you are using the SDS011 for air quality measurements at home, you
probably don't need the from every second. So, in order to increase
its lifespan, you can set it to work in a cycle: sleep for some number
of minutes, wake up for 30s, go back to sleep. To do that, you can use
`sds011cmd`:

```
$ GOOS=linux GOARCH=arm go build  ./go/cmd/sds011cmd && rsync --progress -v -e ssh sds011 pi@pi:
$ ssh pi@pi './sds011 set_cycle 30' # set it to 30 minutes, the maximum
$  ssh pi@pi './sds011cmd  cycle'
30
```

The you can read it with `sds011`. Note that you probably should not
be trying to read and sends commands to the sensor at the same time.

# Advanced

If you need something more complex, you should be able to write a Go
program. Take a look at
https://godoc.org/github.com/ryszard/sds011/go/sds011 for an idea of what you can do.

# License

[Apache 2.0](https://www.tldrlegal.com/l/apache2), please see the file
LICENSE.