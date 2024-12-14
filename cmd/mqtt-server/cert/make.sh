#!/bin/bash
#生成私钥 ssl.key：
openssl genrsa -out ca.key 2048


openssl req -x509 -new -nodes -key ca.key -sha256 -days 365000 -out ca.pem


openssl x509 -in ca.pem -noout -text


openssl genrsa -out mqtt.key 2048


cat << EOF > openssl.cnf
[req]
default_bits  = 2048
distinguished_name = req_distinguished_name
req_extensions = req_ext
x509_extensions = v3_req
prompt = no
[req_distinguished_name]
countryName = CN
stateOrProvinceName = Zhejiang
localityName = Hangzhou
organizationName = mqtt
commonName = Server certificate
[req_ext]
subjectAltName = @alt_names
[v3_req]
subjectAltName = @alt_names
[alt_names]
IP.1 = 0.0.0.0
IP.2 = 127.0.0.1
DNS.1 = iot-mesh.local
EOF

openssl req -new -key mqtt.key -config openssl.cnf -out mqtt.csr
openssl x509 -req -in mqtt.csr -CA ca.pem -CAkey ca.key -CAcreateserial -out mqtt.pem -days 365000 -sha256 -extensions v3_req -extfile openssl.cnf
openssl x509 -in mqtt.pem -noout -text
openssl verify -CAfile ca.pem mqtt.pem