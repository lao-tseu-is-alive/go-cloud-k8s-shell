GITHUB_TOKEN=PUT_YOUR_OWN_GITHUB_TOKEN_HERE COSIGN_PASSWORD=$(openssl rand -base64 32) cosign generate-key-pair github://lao-tseu-is-alive/go-cloud-k8s-shell
# this command will create the above secrets in your github repository secrets
# and will output :
#   Password written to COSIGN_PASSWORD github actions secret
#   Private key written to COSIGN_PRIVATE_KEY github actions secret
#   Public key written to COSIGN_PUBLIC_KEY github actions secret
# and write the Public key in ./cosign.pub
