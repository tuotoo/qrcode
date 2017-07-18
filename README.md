# QR Code decoder by golang

The project is being developed,Not need zbar.

# PLAN

1. 动态二值化
2. 提升图片扫描的速度:OK
3. 修复大version时丢失行的bug
4. 容错码纠正数据
5. 数据编码方式
    Numbert
    alphanumeric
    8-bit byte: OK
    Kanji

# pprof

    $ cd dev

    $ go build

    $ ./dev

    $ go tool pprof dev cpu-profile.prof

    > web
