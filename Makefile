# Go parameters
GOCMD=go
MAINFILE=src/main.go

build:
	$(GOCMD) build $(MAINFILE)

run:
	$(GOCMD) run $(MAINFILE)
