#!/bin/sh

# generate self-signed CA cert with RFC9500 RSA key
openssl req \
    -x509 \
    -key testRSA2048.key \
    -out rootcert.pem \
    -sha256 \
    -days 3650 \
    -nodes \
    -subj "/C=XX/ST=StateName/L=CityName/O=ExampleCo/OU=ExampleUnit/CN=example.org" \
    -addext "subjectAltName=URI:spiffe://example.org,DNS:workload.example.org" \
    -addext "basicConstraints=critical,CA:TRUE,pathlen:0" \
    -addext "keyUsage=critical,digitalSignature,keyCertSign,cRLSign"

# generate leaf workload cert
openssl req \
    -newkey rsa:2048 \
    -keyout leafkey.pem \
    -x509 \
    -out leafsvid.pem \
    -sha256 \
    -days 3650 \
    -nodes \
    -subj "/C=XX/ST=StateName/L=CityName/O=ExampleCo/OU=ExampleUnit/CN=workload.example.org" \
    -addext "subjectAltName=URI:spiffe://example.org/workload,DNS:workload.example.org" \
    -addext "basicConstraints=critical,CA:FALSE" \
    -addext "keyUsage=digitalSignature" \
    -addext "extendedKeyUsage=serverAuth,clientAuth" \
    -CA rootcert.pem \
    -CAkey testRSA2048.key

# generate leaf server cert
openssl req \
    -newkey rsa:2048 \
    -keyout serverkey.pem \
    -x509 \
    -out serversvid.pem \
    -sha256 \
    -days 3650 \
    -nodes \
    -subj "/C=XX/ST=StateName/L=CityName/O=ExampleCo/OU=ExampleUnit/CN=server.example.org" \
    -addext "subjectAltName=URI:spiffe://example.org/server,DNS:server.example.org" \
    -addext "basicConstraints=critical,CA:FALSE" \
    -addext "keyUsage=digitalSignature" \
    -addext "extendedKeyUsage=serverAuth,clientAuth" \
    -CA rootcert.pem \
    -CAkey testRSA2048.key
