#!/bin/sh
# Parts of this script is copied from https://github.com/superseriousbusiness/gotosocial/blob/main/scripts/auth_flow.sh

set -eux

sudo rm -f /tmp/tweet_fan_out_metrics

# Build ap_benchmark binary
make clean || true
make build

SERVER_URL="${SERVER_URL}"
SERVER_URL_SECOND="${SERVER_URL_SECOND}"

# parse domain name from SERVER_URL
DOMAIN=$(echo ${SERVER_URL} | awk -F[/:] '{print $4}')
DOMAIN_SECOND=$(echo ${SERVER_URL_SECOND} | awk -F[/:] '{print $4}')

# generate fake usernames
USERNAME=$(tr -dc a-z0-9 </dev/urandom | head -c 10; echo)

REDIRECT_URI="${SERVER_URL}"
REDIRECT_URI_SECOND="${SERVER_URL_SECOND}"
CLIENT_NAME="Test Application Name"
REGISTRATION_REASON="Testing whether or not this dang diggity thing works!"
REGISTRATION_USERNAME="${USERNAME}"
REGISTRATION_EMAIL="${USERNAME}@${DOMAIN}"
REGISTRATION_PASSWORD="very good password 123"
REGISTRATION_AGREEMENT="true"
REGISTRATION_LOCALE="en"
REGISTRATION_USERNAME_SECOND="${USERNAME}"
REGISTRATION_EMAIL_SECOND="${USERNAME}@${DOMAIN_SECOND}"

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

# Note: Refactor code block below is a copy of code block above, refactor and make it one function.
# Step 1: create the app to register the new account
CREATE_APP_RESPONSE_SECOND=$(curl --fail -s -X POST -F "client_name=${CLIENT_NAME}" -F "redirect_uris=${REDIRECT_URI_SECOND}" "${SERVER_URL_SECOND}/api/v1/apps")
CLIENT_ID_SECOND=$(echo "${CREATE_APP_RESPONSE_SECOND}" | jq -r .client_id)
CLIENT_SECRET_SECOND=$(echo "${CREATE_APP_RESPONSE_SECOND}" | jq -r .client_secret)
echo "Obtained client_id: ${CLIENT_ID_SECOND} and client_secret: ${CLIENT_SECRET_SECOND}"
# Step 2: obtain a code for that app
APP_CODE_RESPONSE_SECOND=$(curl --fail -s -X POST -F "scope=read" -F "grant_type=client_credentials" -F "client_id=${CLIENT_ID_SECOND}" -F "client_secret=${CLIENT_SECRET_SECOND}" -F "redirect_uri=${REDIRECT_URI_SECOND}" "${SERVER_URL_SECOND}/oauth/token")
APP_ACCESS_TOKEN_SECOND=$(echo "${APP_CODE_RESPONSE_SECOND}" | jq -r .access_token)
echo "Obtained app access token: ${APP_ACCESS_TOKEN_SECOND}"
# Step 3: use the code to register a new account
ACCOUNT_REGISTER_RESPONSE_SECOND=$(curl --fail -s -H "Authorization: Bearer ${APP_ACCESS_TOKEN_SECOND}" -F "reason=${REGISTRATION_REASON}" -F "email=${REGISTRATION_EMAIL_SECOND}" -F "username=${REGISTRATION_USERNAME_SECOND}" -F "password=${REGISTRATION_PASSWORD}" -F "agreement=${REGISTRATION_AGREEMENT}" -F "locale=${REGISTRATION_LOCALE}" "${SERVER_URL_SECOND}/api/v1/accounts")
USER_ACCESS_TOKEN_SECOND=$(echo "${ACCOUNT_REGISTER_RESPONSE_SECOND}" | jq -r .access_token)
echo "Obtained user access token: ${USER_ACCESS_TOKEN_SECOND}"
# # Step 4: verify the returned access token
curl -s -H "Authorization: Bearer ${USER_ACCESS_TOKEN_SECOND}" "${SERVER_URL_SECOND}/api/v1/accounts/verify_credentials" | jq


# Run test for this user
./main -client-id ${CLIENT_ID} -client-secret ${CLIENT_SECRET} -access-token ${USER_ACCESS_TOKEN} -instance ${SERVER_URL} -userid-first ${REGISTRATION_EMAIL} -client-id-second ${CLIENT_ID_SECOND} -client-secret-second ${CLIENT_SECRET_SECOND} -access-token-second ${USER_ACCESS_TOKEN_SECOND} -instance-second ${SERVER_URL_SECOND} -userid-second ${REGISTRATION_EMAIL_SECOND} -load 100 -parallel true
cat /tmp/tweet_fan_out_metrics
cat /tmp/tweet_fan_out_metrics | awk '{print $15}' | sed 's/\ms$//' | gnuplot -p -e 'set xtics 1; set ylabel "duration (ms)"; set xlabel "Parallel request no"; plot "/dev/stdin" using 1 title "Duration" w linespoints pt 7'
