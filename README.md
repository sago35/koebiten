# koebiten

koebiten is a miniten clone that runs on TinyGo.

* https://github.com/eihigh/miniten

## games/flappygopher

### run with koebiten

![](./images/flappygopher.jpg)

For now, koebiten only works on [zero-kb02](https://github.com/sago35/keyboards). It needs some improvements to run in a more general environment.  

```
$ tinygo flash --target waveshare-rp2040-zero --size short ./games/flappygopher
```

### run with miniten

![](./images/flappygopher_miniten.png)

The same source code mentioned above can be run on miniten and on a computer.  
When run in a non-TinyGo environment, koebiten simply calls miniten's functions.  

```
$ go run ./games/flappygopher
```

## link

* https://github.com/eihigh/miniten
* https://github.com/sago35/keyboards
