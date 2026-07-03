WASM := plugin.wasm
NDP := navirpc.ndp

build:
	cd plugin && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$(WASM) .

# fallback if std-go wasm misbehaves in navidrome (needs tinygo installed)
tinygo:
	cd plugin && tinygo build -opt=2 -scheduler=none -no-debug -target wasip1 -buildmode=c-shared -o ../$(WASM) .

package: build
	rm -f $(NDP)
	python3 -c "import zipfile; z=zipfile.ZipFile('$(NDP)','w',zipfile.ZIP_DEFLATED); z.write('$(WASM)','plugin.wasm'); z.write('manifest.json','manifest.json'); z.close()"

test:
	go test ./internal/...

clean:
	rm -f $(WASM) $(NDP)

.PHONY: build tinygo package test clean
