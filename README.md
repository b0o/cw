cw: colorwin
------------

This program is nearly useless: all it does is create a blank window with a
given color. Use it with a tiling WM like Sway to create blank spaces in your
layout or to hold an empty workspace open.

### Usage:

```
cw [color]

Where color is a hexadecimal color code, e.g. 'ff0000'.

Short color codes are supported, e.g. 'f7b' expands to 'ff777bb'.
```

### Installation

```
$ go get -u github.com/b0o/cw
```


### TODO

- [ ] Support transparency
- [ ] Basic keybinds

## License

&copy;2021-2022 Maddison Hellstrom

Released under the MIT License.
