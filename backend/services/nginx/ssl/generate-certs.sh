#!/bin/bash
# Generate self-signed SSL certificates for Context Services

set -e

echo "🔐 Generating SSL certificates for Context Services..."

# Create SSL directory if it doesn't exist
mkdir -p /etc/nginx/ssl

# Generate private key
openssl genrsa -out /etc/nginx/ssl/context-services.key 4096

# Generate certificate signing request
openssl req -new -key /etc/nginx/ssl/context-services.key -out /etc/nginx/ssl/context-services.csr -subj "/C=US/ST=CA/L=San Francisco/O=Clinical Synthesis Hub/OU=Context Services/CN=context-services.local"

# Generate self-signed certificate
openssl x509 -req -days 365 -in /etc/nginx/ssl/context-services.csr -signkey /etc/nginx/ssl/context-services.key -out /etc/nginx/ssl/context-services.crt

# Set proper permissions
chmod 600 /etc/nginx/ssl/context-services.key
chmod 644 /etc/nginx/ssl/context-services.crt

echo "✅ SSL certificates generated successfully"
echo "   Certificate: /etc/nginx/ssl/context-services.crt"
echo "   Private Key: /etc/nginx/ssl/context-services.key"

# Create DH parameters for enhanced security
openssl dhparam -out /etc/nginx/ssl/dhparam.pem 2048
chmod 644 /etc/nginx/ssl/dhparam.pem

echo "✅ DH parameters generated"

echo "🎉 SSL setup complete!"