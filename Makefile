smoketest: FORCE
	mkdir -p out
	tinygo build -o ./out/all.zero-kb02.uf2          --target ./targets/zero-kb02.json --size short ./games/all/
	tinygo build -o ./out/flappygopher.zero-kb02.uf2 --target ./targets/zero-kb02.json --size short ./games/flappygopher/
	tinygo build -o ./out/jumpingopher.zero-kb02.uf2 --target ./targets/zero-kb02.json --size short ./games/jumpingopher/
	tinygo build -o ./out/blocks.zero-kb02.uf2       --target ./targets/zero-kb02.json --size short ./games/blocks/
	tinygo build -o ./out/all.gopher-badge.uf2          --target gopher-badge --size short ./games/all/
	tinygo build -o ./out/flappygopher.gopher-badge.uf2 --target gopher-badge --size short ./games/flappygopher/
	tinygo build -o ./out/jumpingopher.gopher-badge.uf2 --target gopher-badge --size short ./games/jumpingopher/
	tinygo build -o ./out/blocks.gopher-badge.uf2       --target gopher-badge --size short ./games/blocks/

FORCE:
