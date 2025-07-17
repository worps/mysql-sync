@echo off

if "%1" == "linux" (
	:: 需要先安装docker
	echo run docker ...
	docker run -it ^
		-v %cd%:/mysrc ^
		golang ^
		cd /mysrc ^
		go build -tags netgo -ldflags '-w -s -extldflags "-static"' -o build/dbdiff main.go
) else (
	go build -ldflags="-w -s -extldflags '-static'" -o ./build/sync.exe .\main.go
)