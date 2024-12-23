BINARY_NAME=limbo

all: 
	cd frontend && npm install
	cd frontend && npm run build
	go mod tidy
	go build -o $(BINARY_NAME) .

fdev:
	cd frontend && npm install
	cd frontend && npm run dev

bdev:
	go mod tidy
	go run .

clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f go.sum
	rm -rf frontend/dist
	rm -rf frontend/node_modules
