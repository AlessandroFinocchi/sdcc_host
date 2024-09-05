rm *.pem

# 1. Generate CA's private key and self-signed certificate
# valid for 1 year, x509 allows to self-sign the certificate
# -nodes doesn't encrypt the private key with a password, so we don't need to communicate it as a manual input
openssl req -x509 -newkey rsa:4096 -days 365 -nodes -keyout ca-key.pem -out ca-cert.pem -subj "/C=IT/ST=Lazio/L=Roma/O=Tor Vergata/OU=SDCC/CN=*.af/emailAddress=alessandro.22082001@gmail.com"

echo "CA's self-signed certificate"
# Display the content of the certificate
openssl x509 -in ca-cert.pem -noout -text

# 2. Generate web server's private key and certificate signing request (CSR)
# This time don't use x509 option, because we want to generate a CSR and not a self-signed certificate
openssl req -newkey rsa:4096 -nodes -keyout server-key.pem -out server-req.pem -subj "/C=IT/ST=Lazio/L=Roma/O=PC Server /OU=Computer/CN=*.server.com/emailAddress=alessandro.22082001@gmail.com"

# 3. Use CA's private key to sign web server's CSR and get back the signed certificate
openssl x509 -req -in server-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -extfile server-ext.cnf

echo "Server's signed certificate"
openssl x509 -in server-cert.pem -noout -text

# 4. Generate client's private key and certificate signing request (CSR)
openssl req -newkey rsa:4096 -nodes -keyout client-key.pem -out client-req.pem -subj "/C=IT/ST=Lazio/L=Roma/O=PC Client/OU=Computer/CN=*.client.com/emailAddress=alessandro.22082001@gmail.com "

# 5. Use CA's private key to sign client's CSR and get back the signed certificate
openssl x509 -req -in client-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out client-cert.pem -extfile client-ext.cnf

echo "Client's signed certificate"
openssl x509 -in client-cert.pem -noout -text
