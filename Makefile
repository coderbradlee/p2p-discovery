GO=go
default: build/seeker
build/caster:
	${GO} build -tags nocgo -o main seeker
