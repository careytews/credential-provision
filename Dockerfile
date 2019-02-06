
FROM fedora:28

RUN dnf install -y openssl
RUN dnf install -y words

RUN mkdir /cred-mgmt
WORKDIR /cred-mgmt

COPY create-vpn-key create-web-key decode destroy-ckms do-create-vpn-key \
  do-create-web-key encode-file encode-key encode-secret generate-key \
  setup-ckms upload-to-storage upload-crl-to-storage download-from-storage create-key \
  create-ca-cert create-cert-request create-cert create-crl \
  find-cert delete-from-storage create-all-crls  revoke-probe-key \
  create-probe-key do-create-probe-key create-vpn-service-key \
  do-create-vpn-service-key revoke-vpn-service-key \
  revoke-all-key revoke-web-key revoke-vpn-key update-index-file /cred-mgmt/
  
COPY credential-provision /cred-mgmt/

COPY client.conf /cred-mgmt/

CMD ["./credential-provision"]

EXPOSE 8080

