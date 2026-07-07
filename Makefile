WASM := plugin.wasm
NDP := navirpc.ndp

build:
	cd plugin && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ../$(WASM) .

package: build
	rm -f $(NDP)
	python3 -c "import zipfile; z=zipfile.ZipFile('$(NDP)','w',zipfile.ZIP_DEFLATED); z.write('$(WASM)','plugin.wasm'); z.write('manifest.json','manifest.json'); z.close()"

test:
	go test ./internal/...

clean:
	rm -f $(WASM) $(NDP)

.PHONY: build package test clean
