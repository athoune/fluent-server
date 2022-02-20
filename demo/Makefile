all: ca-key.pem server-key.pem

ca-key.pem:
	cfssl gencert -initca ca.csr.json | cfssljson -bare ca

server-key.pem:
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=ca-config.json \
		-profile=server \
		server.json | cfssljson -bare server

clean-ca:
	rm -f ca.csr ca.crt ca.key

clean-server:
	rm -f server.csr server.crt server.key

clean: clean-ca clean-server
