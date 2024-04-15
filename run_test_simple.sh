#!/bin/sh
# Parts of this script is copied from https://github.com/superseriousbusiness/gotosocial/blob/main/scripts/auth_flow.sh

set -eux

# Build ap_benchmark binary
make clean || true
make build

SERVER_URL=${SERVER_URL:-"http://localhost:8080"}

# parse domain name from SERVER_URL
DOMAIN=$(echo ${SERVER_URL} | awk -F[/:] '{print $4}')
# generate fake usernames
USERNAME=$(tr -dc a-z0-9 </dev/urandom | head -c 10; echo)

REDIRECT_URI="${SERVER_URL}"
CLIENT_NAME="Test Application Name"
REGISTRATION_REASON="Testing whether or not this dang diggity thing works!"
REGISTRATION_USERNAME="${USERNAME}"
REGISTRATION_EMAIL="${USERNAME}@${DOMAIN}"
REGISTRATION_PASSWORD="very good password 123"
REGISTRATION_AGREEMENT="true"
REGISTRATION_LOCALE="en"
LOAD_REQUESTS=FOO="${3:-1}"

# Step 1: create the app to register the new account
CREATE_APP_RESPONSE=$(curl --fail -s -X POST -F "client_name=${CLIENT_NAME}" -F "redirect_uris=${REDIRECT_URI}" "${SERVER_URL}/api/v1/apps")
CLIENT_ID=$(echo "${CREATE_APP_RESPONSE}" | jq -r .client_id)
CLIENT_SECRET=$(echo "${CREATE_APP_RESPONSE}" | jq -r .client_secret)
echo "Obtained client_id: ${CLIENT_ID} and client_secret: ${CLIENT_SECRET}"

# Step 2: obtain a code for that app
APP_CODE_RESPONSE=$(curl --fail -s -X POST -F "scope=read" -F "grant_type=client_credentials" -F "client_id=${CLIENT_ID}" -F "client_secret=${CLIENT_SECRET}" -F "redirect_uri=${REDIRECT_URI}" "${SERVER_URL}/oauth/token")
APP_ACCESS_TOKEN=$(echo "${APP_CODE_RESPONSE}" | jq -r .access_token)
echo "Obtained app access token: ${APP_ACCESS_TOKEN}"

# Step 3: use the code to register a new account
ACCOUNT_REGISTER_RESPONSE=$(curl --fail -s -H "Authorization: Bearer ${APP_ACCESS_TOKEN}" -F "reason=${REGISTRATION_REASON}" -F "email=${REGISTRATION_EMAIL}" -F "username=${REGISTRATION_USERNAME}" -F "password=${REGISTRATION_PASSWORD}" -F "agreement=${REGISTRATION_AGREEMENT}" -F "locale=${REGISTRATION_LOCALE}" "${SERVER_URL}/api/v1/accounts")
USER_ACCESS_TOKEN=$(echo "${ACCOUNT_REGISTER_RESPONSE}" | jq -r .access_token)
echo "Obtained user access token: ${USER_ACCESS_TOKEN}"

# # Step 4: verify the returned access token
curl -s -H "Authorization: Bearer ${USER_ACCESS_TOKEN}" "${SERVER_URL}/api/v1/accounts/verify_credentials" | jq

# Run test for this user
./main -client-id ${CLIENT_ID} -client-secret ${CLIENT_SECRET} -access-token ${USER_ACCESS_TOKEN} -instance ${SERVER_URL}
