all: protogen protogents

protogen:
	export CWD=$$(pwd) && \
	cd $${GOPATH}/src && \
	protowrap \
		-I $${GOPATH}/src \
		--go_out=$${GOPATH}/src \
		--proto_path $${GOPATH}/src \
		--print_structure \
		--only_specified_files \
		$${CWD}/*.proto

protogents:
	npm run gen-proto
