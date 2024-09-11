# koebiten

Koebiten is a package for making simple games.
It was initially created based on a package called miniten.

* https://github.com/eihigh/miniten

## games/flappygopher

### run with koebiten

![](./images/flappygopher.jpg)

For now, koebiten only works on [zero-kb02](https://github.com/sago35/keyboards) and [macropad-rp2040](https://learn.adafruit.com/adafruit-macropad-rp2040). It needs some improvements to run in a more general environment.  

```
$ tinygo flash --target waveshare-rp2040-zero --size short ./games/flappygopher
```

## games/jumpingopher

### run with koebiten

![](./images/jumpingopher.jpg)

```
$ tinygo flash --target waveshare-rp2040-zero --size short ./games/jumpingopher
```

more info : [./games/jumpingopher](./games/jumpingopher)

## link

* https://github.com/eihigh/miniten
* https://github.com/sago35/keyboards
