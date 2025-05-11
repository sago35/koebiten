smoketest: FORCE
	mkdir -p out
	tinygo build -o ./out/all.zero-kb02.uf2          --size short --target ./targets/zero-kb02.json        ./games/all/
	tinygo build -o ./out/flappygopher.zero-kb02.uf2 --size short --target ./targets/zero-kb02.json        ./games/flappygopher/
	tinygo build -o ./out/jumpingopher.zero-kb02.uf2 --size short --target ./targets/zero-kb02.json        ./games/jumpingopher/
	tinygo build -o ./out/blocks.zero-kb02.uf2       --size short --target ./targets/zero-kb02.json        ./games/blocks/
	tinygo build -o ./out/example_GeoM.zero-kb02.uf2 --size short --target ./targets/zero-kb02.json        ./examples/GeoM/
	tinygo build -o ./out/all.gopher-badge.uf2       --size short --target gopher-badge                    ./games/all/
	tinygo build -o ./out/all.pybadge.uf2            --size short --target pybadge                         ./games/all/
	tinygo build -o ./out/all.wioterminal.uf2        --size short --target wioterminal                     ./games/all/
	tinygo build -o ./out/all.macropad-rp2040.uf2    --size short --target macropad-rp2040                 ./games/all/
	tinygo build -o ./out/all.gopher-board-i2c.uf2   --size short --target ./targets/gopher-board-i2c.json ./games/all/
	tinygo build -o ./out/all.gopher-board-spi.uf2   --size short --target ./targets/gopher-board-spi.json ./games/all/
	tinygo build -o ./out/all.wasm                   --size short --target wasm --no-debug --panic trap    ./games/all/
	tinygo build -o ./out/goradius.zero-kb02.uf2 --size short --target ./targets/zero-kb02.json            ./games/goradius/

FORCE:
