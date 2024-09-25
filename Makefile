smoketest: FORCE
	mkdir -p out
	tinygo build -o ./out/flappygopher.uf2 --target waveshare-rp2040-zero --size short ./games/flappygopher/
	tinygo build -o ./out/jumpingopher.uf2 --target waveshare-rp2040-zero --size short ./games/jumpingopher/
	tinygo build -o ./out/blocks.uf2       --target waveshare-rp2040-zero --size short ./games/blocks/

FORCE:
