GO=go
default: ./
seeker:
	${GO} build default -tags nocgo -o seeker
