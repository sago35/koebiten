smoketest: FORCE
	mkdir -p out
	tinygo build -o ./out/flappygopher.uf2 --target waveshare-rp2040-zero --size short ./games/flappygopher/
	go build -o ./out/flappygopher ./games/flappygopher/

FORCE:
